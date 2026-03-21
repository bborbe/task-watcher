// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package watcher

import (
	"context"
	"log/slog"
	"slices"

	"github.com/bborbe/errors"
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

// Watcher watches the vault tasks directory and notifies on matching task changes.
//
//counterfeiter:generate -o ../../mocks/watcher.go --fake-name FakeWatcher . Watcher
type Watcher interface {
	Watch(ctx context.Context) error
}

// NewWatcher returns a Watcher that watches cfg.VaultPath and calls notifier for matching tasks.
func NewWatcher(cfg config.Config, notifier notify.Notifier) Watcher {
	storageConfig := &storage.Config{
		TasksDir: "24 Tasks",
	}
	return &watcher{
		config:      cfg,
		notifier:    notifier,
		watchOp:     ops.NewWatchOperation(),
		taskStorage: storage.NewStorage(storageConfig),
	}
}

type watcher struct {
	config      config.Config
	notifier    notify.Notifier
	watchOp     ops.WatchOperation
	taskStorage taskReader
}

// Watch starts watching the vault tasks directory until ctx is cancelled.
func (w *watcher) Watch(ctx context.Context) error {
	targets := []ops.WatchTarget{{
		VaultPath: w.config.VaultPath,
		VaultName: "vault",
		WatchDirs: []string{"24 Tasks"},
	}}
	return w.watchOp.Execute(ctx, targets, func(event ops.WatchEvent) error {
		if event.Event != "created" && event.Event != "modified" {
			return nil
		}
		return w.handleEvent(ctx, event)
	})
}

func (w *watcher) handleEvent(ctx context.Context, event ops.WatchEvent) error {
	task, err := w.taskStorage.ReadTask(ctx, w.config.VaultPath, domain.TaskID(event.Name))
	if err != nil {
		slog.Debug("skip unreadable task", "name", event.Name, "error", err)
		return nil
	}
	if task.Assignee == "" || task.Status == "" || task.Phase == nil {
		return nil
	}
	if task.Assignee != w.config.Assignee {
		return nil
	}
	if !containsString(w.config.Statuses, string(task.Status)) {
		return nil
	}
	if !containsString(w.config.Phases, string(*task.Phase)) {
		return nil
	}
	if err := w.notifier.Notify(ctx, notify.Notification{
		TaskName: task.Name,
		Phase:    string(*task.Phase),
		Assignee: task.Assignee,
	}); err != nil {
		return errors.Wrapf(ctx, err, "notify task %s phase %s", task.Name, string(*task.Phase))
	}
	return nil
}

func containsString(slice []string, s string) bool {
	return slices.Contains(slice, s)
}
