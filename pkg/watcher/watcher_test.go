// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package watcher_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mocknotify "github.com/bborbe/task-watcher/mocks"
	"github.com/bborbe/task-watcher/pkg/config"
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
			Assignee: "Alice",
			Statuses: []string{"in_progress"},
			Phases:   []string{"planning"},
		}
		fakeNotifier = &mocknotify.FakeNotifier{}
		w = watcher.NewWatcher(cfg, fakeNotifier)

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

	Context("when a matching task is created", func() {
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

	Context("when a task has the wrong assignee", func() {
		It("does not call the notifier", func() {
			writeTask(tasksDir, "other-task", "Bob", "in_progress", "planning")
			Consistently(func() int {
				return fakeNotifier.NotifyCallCount()
			}, "300ms", "20ms").Should(Equal(0))
		})
	})

	Context("when a task has the wrong status", func() {
		It("does not call the notifier", func() {
			writeTask(tasksDir, "todo-task", "Alice", "todo", "planning")
			Consistently(func() int {
				return fakeNotifier.NotifyCallCount()
			}, "300ms", "20ms").Should(Equal(0))
		})
	})

	Context("when a task has the wrong phase", func() {
		It("does not call the notifier", func() {
			writeTask(tasksDir, "other-phase-task", "Alice", "in_progress", "review")
			Consistently(func() int {
				return fakeNotifier.NotifyCallCount()
			}, "300ms", "20ms").Should(Equal(0))
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
			Assignee: "Alice",
			Statuses: []string{"in_progress"},
			Phases:   []string{"planning"},
		}
		fakeNotifier = &mocknotify.FakeNotifier{}
		w = watcher.NewWatcher(cfg, fakeNotifier)

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
