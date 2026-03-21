// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"log/slog"
	"sync"
)

// NewDryRunNotifier returns a Notifier that logs notifications instead of sending HTTP requests.
func NewDryRunNotifier() Notifier {
	return &dryRunNotifier{
		seen: make(map[string]struct{}),
	}
}

type dryRunNotifier struct {
	seen map[string]struct{}
	mu   sync.Mutex
}

func (d *dryRunNotifier) Notify(_ context.Context, n Notification) error {
	key := n.TaskName + ":" + n.Phase

	d.mu.Lock()
	_, exists := d.seen[key]
	if exists {
		d.mu.Unlock()
		return nil
	}
	d.seen[key] = struct{}{}
	d.mu.Unlock()

	slog.Info(
		"dry-run notification",
		"taskName",
		n.TaskName,
		"phase",
		n.Phase,
		"assignee",
		n.Assignee,
	)
	return nil
}
