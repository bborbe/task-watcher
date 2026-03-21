// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bborbe/errors"
)

// NewDryRunNotifier returns a Notifier that logs notifications instead of sending HTTP requests.
func NewDryRunNotifier(webhookURL string, dedupTTL time.Duration) Notifier {
	return &dryRunNotifier{
		webhookURL: webhookURL,
		dedupTTL:   dedupTTL,
		seen:       make(map[string]time.Time),
	}
}

type dryRunNotifier struct {
	webhookURL string
	dedupTTL   time.Duration
	seen       map[string]time.Time
	mu         sync.Mutex
}

func (d *dryRunNotifier) Notify(ctx context.Context, n Notification) error {
	key := n.TaskName + ":" + n.Phase

	d.mu.Lock()
	lastSent, exists := d.seen[key]
	if exists && time.Since(lastSent) < d.dedupTTL {
		d.mu.Unlock()
		return nil
	}
	d.seen[key] = time.Now()
	d.mu.Unlock()

	body, err := json.Marshal(n)
	if err != nil {
		return errors.Wrapf(ctx, err, "marshal notification")
	}
	slog.Info("dry-run: would send webhook",
		"method", "POST",
		"url", d.webhookURL,
		"header", "Content-Type: application/json",
		"body", string(body),
	)
	return nil
}

// NewDryRunOpenClawNotifier returns a Notifier that logs the OpenClaw payload instead of sending HTTP requests.
func NewDryRunOpenClawNotifier(webhookURL string, token string, dedupTTL time.Duration) Notifier {
	return &dryRunOpenClawNotifier{
		webhookURL: webhookURL,
		token:      token,
		dedupTTL:   dedupTTL,
		seen:       make(map[string]time.Time),
	}
}

type dryRunOpenClawNotifier struct {
	webhookURL string
	token      string
	dedupTTL   time.Duration
	seen       map[string]time.Time
	mu         sync.Mutex
}

func (d *dryRunOpenClawNotifier) Notify(ctx context.Context, n Notification) error {
	key := n.TaskName + ":" + n.Phase

	d.mu.Lock()
	lastSent, exists := d.seen[key]
	if exists && time.Since(lastSent) < d.dedupTTL {
		d.mu.Unlock()
		return nil
	}
	d.seen[key] = time.Now()
	d.mu.Unlock()

	payload := openClawPayload{
		Text: fmt.Sprintf(
			"Task watcher: %s task changed (task: %s, phase: %s)",
			n.Assignee,
			n.TaskName,
			n.Phase,
		),
		Mode: "now",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(ctx, err, "marshal openclaw payload")
	}
	slog.Info("dry-run: would send openclaw webhook",
		"method", "POST",
		"url", d.webhookURL,
		"header", "Content-Type: application/json",
		"header", "Authorization: Bearer "+d.token,
		"body", string(body),
	)
	return nil
}
