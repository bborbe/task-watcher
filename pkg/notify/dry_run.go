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

	"github.com/bborbe/errors"
)

// NewDryRunNotifier returns a Notifier that logs notifications instead of sending HTTP requests.
func NewDryRunNotifier(webhookURL string) Notifier {
	return &dryRunNotifier{
		webhookURL: webhookURL,
		seen:       make(map[string]struct{}),
	}
}

type dryRunNotifier struct {
	webhookURL string
	seen       map[string]struct{}
	mu         sync.Mutex
}

func (d *dryRunNotifier) Notify(ctx context.Context, n Notification) error {
	key := n.TaskName + ":" + n.Phase

	d.mu.Lock()
	_, exists := d.seen[key]
	if exists {
		d.mu.Unlock()
		return nil
	}
	d.seen[key] = struct{}{}
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
func NewDryRunOpenClawNotifier(webhookURL string, token string) Notifier {
	return &dryRunOpenClawNotifier{
		webhookURL: webhookURL,
		token:      token,
		seen:       make(map[string]struct{}),
	}
}

type dryRunOpenClawNotifier struct {
	webhookURL string
	token      string
	seen       map[string]struct{}
	mu         sync.Mutex
}

func (d *dryRunOpenClawNotifier) Notify(ctx context.Context, n Notification) error {
	key := n.TaskName + ":" + n.Phase

	d.mu.Lock()
	_, exists := d.seen[key]
	if exists {
		d.mu.Unlock()
		return nil
	}
	d.seen[key] = struct{}{}
	d.mu.Unlock()

	payload := openClawPayload{
		Name: "task-watcher",
		Message: fmt.Sprintf(
			"Task update: %s. Assignee: %s. Phase: %s.",
			n.TaskName,
			n.Assignee,
			n.Phase,
		),
		WakeMode: "now",
		Deliver:  false,
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
