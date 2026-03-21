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
			VaultPath: vaultDir,
			Assignee:  "Alice",
			Statuses:  []string{"in_progress"},
			Phases:    []string{"planning"},
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
