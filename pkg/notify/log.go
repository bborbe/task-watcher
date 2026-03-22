// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// NewLogNotifier returns a Notifier that logs notifications to stdout instead of sending HTTP requests.
// Use this as the action type for debugging or dry-run observation.
func NewLogNotifier(dedupTTL time.Duration) Notifier {
	return &logNotifier{
		dedupTTL: dedupTTL,
		seen:     make(map[string]time.Time),
	}
}

type logNotifier struct {
	dedupTTL time.Duration
	seen     map[string]time.Time
	mu       sync.Mutex
}

func (l *logNotifier) Notify(_ context.Context, notification Notification) error {
	key := notification.TaskName + ":" + notification.Phase

	l.mu.Lock()
	lastSent, exists := l.seen[key]
	if exists && time.Since(lastSent) < l.dedupTTL {
		l.mu.Unlock()
		slog.Debug("log notifier skipped (duplicate within TTL)",
			"task", notification.TaskName,
			"phase", notification.Phase,
			"ttl", l.dedupTTL,
			"lastSent", lastSent,
		)
		return nil
	}
	l.seen[key] = time.Now()
	l.mu.Unlock()

	slog.Info("log notifier: task event",
		"task", notification.TaskName,
		"phase", notification.Phase,
		"assignee", notification.Assignee,
	)
	return nil
}
