// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package publish

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bborbe/cqrs/base"
	"github.com/bborbe/errors"
	"github.com/bborbe/notification/command/notification"
	libtime "github.com/bborbe/time"
)

//counterfeiter:generate -o ../../mocks/notification_publish_command_sender.go --fake-name NotificationPublishCommandSender github.com/bborbe/notification/command/notification.NotificationPublishCommandSender

// Publisher publishes one notification per matched task event, suppressing
// repeats of the same task and phase inside its window.
type Publisher interface {
	Publish(ctx context.Context, event TaskEvent) error
}

// NewPublisher returns a Publisher that publishes through sender and keeps its
// own dedup window (one instance per watcher entry, using that entry's TTL).
// The window is measured on currentDateTimeGetter, so expiry is testable
// without sleeping.
func NewPublisher(
	sender notification.NotificationPublishCommandSender,
	topicPrefix base.TopicPrefix,
	dedupTTL time.Duration,
	currentDateTimeGetter libtime.CurrentDateTimeGetter,
) Publisher {
	return &publisher{
		sender:                sender,
		topicPrefix:           topicPrefix,
		dedupTTL:              dedupTTL,
		currentDateTimeGetter: currentDateTimeGetter,
		seen:                  make(map[string]libtime.DateTime),
	}
}

type publisher struct {
	sender                notification.NotificationPublishCommandSender
	topicPrefix           base.TopicPrefix
	dedupTTL              time.Duration
	currentDateTimeGetter libtime.CurrentDateTimeGetter
	seen                  map[string]libtime.DateTime
	mu                    sync.Mutex
}

// Publish builds, validates and sends the notification command for one matched
// task event. A repeat of the same task and phase inside the window is
// suppressed and returns nil. The window is consumed by the attempt itself, so
// a failed send is not retried inside the same window.
func (p *publisher) Publish(ctx context.Context, event TaskEvent) error {
	now := p.currentDateTimeGetter.Now()
	if p.isDuplicate(event, now) {
		return nil
	}

	cmd, err := BuildCommand(ctx, event)
	if err != nil {
		return errors.Wrapf(ctx, err, "build publish notification command")
	}
	if err := cmd.Validate(ctx); err != nil {
		return errors.Wrapf(ctx, err, "validate publish notification command")
	}

	slog.Info("publish notification",
		"topic_prefix", p.topicPrefix.String(),
		"task", event.TaskName,
		"phase", event.Phase,
		"type", cmd.Type.String(),
	)

	if err := p.sender.SendPublishNotificationCommand(ctx, cmd); err != nil {
		return errors.Wrapf(ctx, err, "send publish notification command")
	}
	return nil
}

// isDuplicate reports whether the task+phase pair was already published inside
// the window. When it was not, the key is stamped with now and entries whose
// age has reached the window are pruned, so the map stays proportional to the
// distinct task+phase pairs seen inside one window. Both the comparison and the
// stamp use the single now read by the caller.
func (p *publisher) isDuplicate(event TaskEvent, now libtime.DateTime) bool {
	key := event.TaskName + ":" + event.Phase

	p.mu.Lock()
	defer p.mu.Unlock()

	if last, exists := p.seen[key]; exists && now.Sub(last).Duration() < p.dedupTTL {
		slog.Debug("skipped (duplicate within TTL)",
			"task", event.TaskName,
			"phase", event.Phase,
			"ttl", p.dedupTTL,
		)
		return true
	}

	p.prune(now)
	p.seen[key] = now
	return false
}

// prune drops every entry whose age has reached the window. The caller holds
// the mutex.
func (p *publisher) prune(now libtime.DateTime) {
	for key, stamped := range p.seen {
		if now.Sub(stamped).Duration() >= p.dedupTTL {
			delete(p.seen, key)
		}
	}
}
