// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package watcher

import (
	"context"
	"log/slog"

	"github.com/bborbe/vault-cli/pkg/domain"
	"github.com/bborbe/vault-cli/pkg/ops"
	"github.com/bborbe/vault-cli/pkg/storage"

	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/notify"
)

// taskReader is a narrow interface for reading a single task by ID.
type taskReader interface {
	ReadTask(ctx context.Context, vaultPath string, taskID domain.TaskID) (*domain.Task, error)
}

// watcherEntry pairs per-watcher filter criteria with a notifier.
type watcherEntry struct {
	name     string
	assignee string
	statuses []string
	phases   []string
	notifier notify.Notifier
}

// Watcher watches the vault tasks directory and notifies on matching task changes.
//
//counterfeiter:generate -o ../../mocks/watcher.go --fake-name FakeWatcher . Watcher
type Watcher interface {
	Watch(ctx context.Context) error
}

// NewWatcher returns a Watcher that watches all configured vaults and fans out
// matching task events to all configured watcher entries.
func NewWatcher(cfg config.Config, notifiers []notify.Notifier) Watcher {
	vaultPaths := make(map[string]string, len(cfg.Vaults))
	taskStorages := make(map[string]taskReader, len(cfg.Vaults))
	targets := make([]ops.WatchTarget, 0, len(cfg.Vaults))

	for _, v := range cfg.Vaults {
		vaultPaths[v.Name] = v.Path
		taskStorages[v.Name] = storage.NewStorage(&storage.Config{TasksDir: v.TasksDir})
		targets = append(targets, ops.WatchTarget{
			VaultPath: v.Path,
			VaultName: v.Name,
			WatchDirs: []string{v.TasksDir},
		})
	}

	entries := make([]watcherEntry, len(cfg.Watchers))
	for i, w := range cfg.Watchers {
		entries[i] = watcherEntry{
			name:     w.Name,
			assignee: w.Assignee,
			statuses: w.Statuses,
			phases:   w.Phases,
			notifier: notifiers[i],
		}
	}

	return &watcher{
		entries:      entries,
		watchOp:      ops.NewWatchOperation(),
		vaultPaths:   vaultPaths,
		taskStorages: taskStorages,
		targets:      targets,
	}
}

type watcher struct {
	entries      []watcherEntry
	watchOp      ops.WatchOperation
	vaultPaths   map[string]string
	taskStorages map[string]taskReader
	targets      []ops.WatchTarget
}

// Watch starts watching all configured vault task directories until ctx is cancelled.
func (w *watcher) Watch(ctx context.Context) error {
	return w.watchOp.Execute(ctx, w.targets, func(event ops.WatchEvent) error {
		if event.Event != "created" && event.Event != "modified" {
			return nil
		}
		return w.handleEvent(ctx, event)
	})
}

func (w *watcher) handleEvent(ctx context.Context, event ops.WatchEvent) error {
	vaultPath, ok := w.vaultPaths[event.Vault]
	if !ok {
		slog.Warn("unknown vault in event", "vault", event.Vault)
		return nil
	}
	taskStorage, ok := w.taskStorages[event.Vault]
	if !ok {
		slog.Warn("no storage for vault", "vault", event.Vault)
		return nil
	}
	task, err := taskStorage.ReadTask(ctx, vaultPath, domain.TaskID(event.Name))
	if err != nil {
		slog.Debug("skip unreadable task", "name", event.Name, "error", err)
		return nil
	}
	if task.Assignee == "" || task.Status == "" || task.Phase == nil {
		return nil
	}

	notification := notify.Notification{
		TaskName: task.Name,
		Phase:    string(*task.Phase),
		Assignee: task.Assignee,
	}

	for _, entry := range w.entries {
		if entry.assignee != "" && task.Assignee != entry.assignee {
			continue
		}
		if len(entry.statuses) > 0 && !containsString(entry.statuses, string(task.Status)) {
			continue
		}
		if len(entry.phases) > 0 && !containsString(entry.phases, string(*task.Phase)) {
			continue
		}
		if err := entry.notifier.Notify(ctx, notification); err != nil {
			slog.Error("notify failed",
				"watcher", entry.name,
				"task", task.Name,
				"phase", string(*task.Phase),
				"error", err,
			)
			// Continue to next watcher — one failure must not block others
		}
	}
	return nil
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
