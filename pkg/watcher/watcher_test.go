// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package watcher_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mocknotify "github.com/bborbe/task-watcher/mocks"
	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/notify"
	"github.com/bborbe/task-watcher/pkg/watcher"
)

func writeTask(dir, name, assignee, status, phase string) {
	content := fmt.Sprintf(
		"---\nassignee: %s\nstatus: %s\nphase: %s\n---\n",
		assignee,
		status,
		phase,
	)
	Expect(os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0600)).To(Succeed())
}

var _ = Describe("Watcher", func() {
	var (
		ctx          context.Context
		cancel       context.CancelFunc
		vaultDir     string
		tasksDir     string
		fakeNotifier *mocknotify.FakeNotifier
		w            watcher.Watcher
		cfg          config.Config
		watchDone    chan error
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var err error
		vaultDir, err = os.MkdirTemp("", "vault-*")
		Expect(err).NotTo(HaveOccurred())
		tasksDir = filepath.Join(vaultDir, "24 Tasks")
		Expect(os.MkdirAll(tasksDir, 0750)).To(Succeed())

		cfg = config.Config{
			Vaults: []config.VaultConfig{
				{Name: "testvault", Path: vaultDir, TasksDir: "24 Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "test", Type: "log"},
			},
		}
		fakeNotifier = &mocknotify.FakeNotifier{}
		w = watcher.NewWatcher(cfg, []notify.Notifier{fakeNotifier})

		watchDone = make(chan error, 1)
		go func() { watchDone <- w.Watch(ctx) }()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		cancel()
		select {
		case <-watchDone:
		case <-time.After(2 * time.Second):
			Fail("Watch did not return after context cancellation")
		}
		Expect(os.RemoveAll(vaultDir)).To(Succeed())
	})

	Context("when a task with complete frontmatter is created", func() {
		It("calls the notifier with the correct task details", func() {
			writeTask(tasksDir, "my-task", "Alice", "in_progress", "planning")
			Eventually(func() int {
				return fakeNotifier.NotifyCallCount()
			}, "500ms", "20ms").Should(Equal(1))
			_, notification := fakeNotifier.NotifyArgsForCall(0)
			Expect(notification.TaskName).To(Equal("my-task"))
			Expect(notification.Phase).To(Equal("planning"))
			Expect(notification.Assignee).To(Equal("Alice"))
		})
	})

	Context("when a task file has missing frontmatter", func() {
		It("does not call the notifier and does not return an error", func() {
			Expect(
				os.WriteFile(filepath.Join(tasksDir, "empty-task.md"), []byte(""), 0600),
			).To(Succeed())
			Consistently(func() int {
				return fakeNotifier.NotifyCallCount()
			}, "300ms", "20ms").Should(Equal(0))
		})
	})

	Context("when context is cancelled", func() {
		It("Watch returns without blocking", func() {
			cancel()
			// AfterEach verifies Watch returns within 2s
		})
	})
})

var _ = Describe("Watcher multi-vault", func() {
	var (
		ctx          context.Context
		cancel       context.CancelFunc
		vault1Dir    string
		vault2Dir    string
		tasks1Dir    string
		tasks2Dir    string
		fakeNotifier *mocknotify.FakeNotifier
		w            watcher.Watcher
		watchDone    chan error
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		var err error
		vault1Dir, err = os.MkdirTemp("", "vault1-*")
		Expect(err).NotTo(HaveOccurred())
		tasks1Dir = filepath.Join(vault1Dir, "24 Tasks")
		Expect(os.MkdirAll(tasks1Dir, 0750)).To(Succeed())

		vault2Dir, err = os.MkdirTemp("", "vault2-*")
		Expect(err).NotTo(HaveOccurred())
		tasks2Dir = filepath.Join(vault2Dir, "Tasks")
		Expect(os.MkdirAll(tasks2Dir, 0750)).To(Succeed())

		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "vault1", Path: vault1Dir, TasksDir: "24 Tasks"},
				{Name: "vault2", Path: vault2Dir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "test", Type: "log"},
			},
		}
		fakeNotifier = &mocknotify.FakeNotifier{}
		w = watcher.NewWatcher(cfg, []notify.Notifier{fakeNotifier})

		watchDone = make(chan error, 1)
		go func() { watchDone <- w.Watch(ctx) }()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		cancel()
		select {
		case <-watchDone:
		case <-time.After(2 * time.Second):
			Fail("Watch did not return after context cancellation")
		}
		Expect(os.RemoveAll(vault1Dir)).To(Succeed())
		Expect(os.RemoveAll(vault2Dir)).To(Succeed())
	})

	It("notifies for tasks from vault1", func() {
		writeTask(tasks1Dir, "task-v1", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return fakeNotifier.NotifyCallCount()
		}, "500ms", "20ms").Should(Equal(1))
		_, notification := fakeNotifier.NotifyArgsForCall(0)
		Expect(notification.TaskName).To(Equal("task-v1"))
	})

	It("notifies for tasks from vault2", func() {
		writeTask(tasks2Dir, "task-v2", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return fakeNotifier.NotifyCallCount()
		}, "500ms", "20ms").Should(Equal(1))
		_, notification := fakeNotifier.NotifyArgsForCall(0)
		Expect(notification.TaskName).To(Equal("task-v2"))
	})

	It("notifies for tasks from both vaults independently", func() {
		writeTask(tasks1Dir, "task-a", "Alice", "in_progress", "planning")
		writeTask(tasks2Dir, "task-b", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return fakeNotifier.NotifyCallCount()
		}, "500ms", "20ms").Should(Equal(2))
	})
})

var _ = Describe("Watcher fan-out", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		vaultDir  string
		tasksDir  string
		watchDone chan error
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var err error
		vaultDir, err = os.MkdirTemp("", "vault-fanout-*")
		Expect(err).NotTo(HaveOccurred())
		tasksDir = filepath.Join(vaultDir, "Tasks")
		Expect(os.MkdirAll(tasksDir, 0750)).To(Succeed())
	})

	AfterEach(func() {
		cancel()
		select {
		case <-watchDone:
		case <-time.After(2 * time.Second):
			Fail("Watch did not return after context cancellation")
		}
		Expect(os.RemoveAll(vaultDir)).To(Succeed())
	})

	startWatcher := func(cfg config.Config, notifiers []notify.Notifier) {
		w := watcher.NewWatcher(cfg, notifiers)
		watchDone = make(chan error, 1)
		go func() { watchDone <- w.Watch(ctx) }()
		time.Sleep(100 * time.Millisecond)
	}

	It("calls both notifiers when task matches both watcher entries", func() {
		fake1 := &mocknotify.FakeNotifier{}
		fake2 := &mocknotify.FakeNotifier{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Type: "log"},
				{Name: "w2", Type: "log"},
			},
		}
		startWatcher(cfg, []notify.Notifier{fake1, fake2})

		writeTask(tasksDir, "both-task", "Alice", "in_progress", "planning")
		Eventually(func() int { return fake1.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))
		Eventually(func() int { return fake2.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))
	})

	It("calls only the matching notifier when task matches only one entry", func() {
		fake1 := &mocknotify.FakeNotifier{}
		fake2 := &mocknotify.FakeNotifier{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Type: "log", Assignee: "Alice"},
				{Name: "w2", Type: "log", Assignee: "Bob"},
			},
		}
		startWatcher(cfg, []notify.Notifier{fake1, fake2})

		writeTask(tasksDir, "alice-task", "Alice", "in_progress", "planning")
		Eventually(func() int { return fake1.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))
		Consistently(
			func() int { return fake2.NotifyCallCount() },
			"200ms",
			"20ms",
		).Should(Equal(0))
	})

	It("calls second notifier even when first notifier returns an error", func() {
		fake1 := &mocknotify.FakeNotifier{}
		fake2 := &mocknotify.FakeNotifier{}
		fake1.NotifyReturns(errors.New("notifier 1 failed"))
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Type: "log"},
				{Name: "w2", Type: "log"},
			},
		}
		startWatcher(cfg, []notify.Notifier{fake1, fake2})

		writeTask(tasksDir, "error-task", "Alice", "in_progress", "planning")
		Eventually(func() int { return fake1.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))
		Eventually(func() int { return fake2.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))
	})

	It("skips entry when task assignee does not match per-watcher assignee filter", func() {
		fake1 := &mocknotify.FakeNotifier{}
		fake2 := &mocknotify.FakeNotifier{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Type: "log", Assignee: "WrongUser"},
				{Name: "w2", Type: "log"},
			},
		}
		startWatcher(cfg, []notify.Notifier{fake1, fake2})

		writeTask(tasksDir, "assignee-task", "Alice", "in_progress", "planning")
		Consistently(
			func() int { return fake1.NotifyCallCount() },
			"200ms",
			"20ms",
		).Should(Equal(0))
		Eventually(func() int { return fake2.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))
	})

	It("skips entry when task phase does not match per-watcher phase filter", func() {
		fake1 := &mocknotify.FakeNotifier{}
		fake2 := &mocknotify.FakeNotifier{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Type: "log", Phases: []string{"execution"}},
				{Name: "w2", Type: "log"},
			},
		}
		startWatcher(cfg, []notify.Notifier{fake1, fake2})

		writeTask(tasksDir, "phase-task", "Alice", "in_progress", "planning")
		Consistently(
			func() int { return fake1.NotifyCallCount() },
			"200ms",
			"20ms",
		).Should(Equal(0))
		Eventually(func() int { return fake2.NotifyCallCount() }, "500ms", "20ms").Should(Equal(1))
	})
})
