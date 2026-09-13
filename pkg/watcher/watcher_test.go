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

	"github.com/bborbe/cqrs/base"
	libtime "github.com/bborbe/time"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mocks "github.com/bborbe/task-watcher/mocks"
	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/publish"
	"github.com/bborbe/task-watcher/pkg/watcher"
)

const testTopicPrefix = base.TopicPrefix("develop")

func writeTask(dir, name, assignee, status, phase string) {
	writeTaskContent(
		dir,
		name,
		fmt.Sprintf("---\nassignee: %s\nstatus: %s\nphase: %s\n---\n", assignee, status, phase),
	)
}

func writeTaskContent(dir, name, content string) {
	Expect(os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0600)).To(Succeed())
}

// newPublisher builds a real publisher on a real clock, exactly as the factory does.
func newPublisher(
	sender *mocks.NotificationPublishCommandSender,
	dedupTTL time.Duration,
) publish.Publisher {
	return publish.NewPublisher(
		sender,
		testTopicPrefix,
		dedupTTL,
		libtime.NewCurrentDateTime(),
	)
}

var _ = Describe("Watcher", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		vaultDir  string
		tasksDir  string
		sender    *mocks.NotificationPublishCommandSender
		w         watcher.Watcher
		cfg       config.Config
		watchDone chan error
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
				{Name: "test", DedupTTL: 5 * time.Minute},
			},
		}
		sender = &mocks.NotificationPublishCommandSender{}
		w = watcher.NewWatcher(cfg, []publish.Publisher{newPublisher(sender, 5*time.Minute)})

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
		It("publishes exactly one agent-escalation command with the task details", func() {
			writeTask(tasksDir, "my-task", "Alice", "in_progress", "planning")
			Eventually(func() int {
				return sender.SendPublishNotificationCommandCallCount()
			}, "500ms", "20ms").Should(Equal(1))

			_, cmd := sender.SendPublishNotificationCommandArgsForCall(0)
			Expect(string(cmd.Type)).To(Equal("agent-escalation"))
			Expect(cmd.Target).To(BeNil())
			Expect(cmd.Message).NotTo(BeEmpty())
			Expect(cmd.Message.String()).To(ContainSubstring("my-task"))
			Expect(cmd.Metadata).To(Equal(map[string]string{
				"taskName": "my-task",
				"phase":    "planning",
				"assignee": "Alice",
				"vault":    "testvault",
			}))
		})
	})

	Context("when a task file has missing frontmatter", func() {
		It("publishes nothing and does not return an error", func() {
			writeTaskContent(tasksDir, "empty-task", "")
			Consistently(func() int {
				return sender.SendPublishNotificationCommandCallCount()
			}, "300ms", "20ms").Should(Equal(0))
		})
	})

	Context("when the guard rejects a task", func() {
		It("publishes nothing for a task with an empty assignee", func() {
			writeTask(tasksDir, "no-assignee", "", "in_progress", "planning")
			Consistently(func() int {
				return sender.SendPublishNotificationCommandCallCount()
			}, "300ms", "20ms").Should(Equal(0))
		})

		It("publishes nothing for a task with an empty status", func() {
			writeTask(tasksDir, "no-status", "Alice", "", "planning")
			Consistently(func() int {
				return sender.SendPublishNotificationCommandCallCount()
			}, "300ms", "20ms").Should(Equal(0))
		})

		It("publishes nothing for a task with no phase", func() {
			writeTaskContent(
				tasksDir,
				"no-phase",
				"---\nassignee: Alice\nstatus: in_progress\n---\n",
			)
			Consistently(func() int {
				return sender.SendPublishNotificationCommandCallCount()
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

var _ = Describe("Watcher publishes once per window", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		vaultDir  string
		tasksDir  string
		sender    *mocks.NotificationPublishCommandSender
		watchDone chan error
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var err error
		vaultDir, err = os.MkdirTemp("", "vault-dedup-*")
		Expect(err).NotTo(HaveOccurred())
		tasksDir = filepath.Join(vaultDir, "Tasks")
		Expect(os.MkdirAll(tasksDir, 0750)).To(Succeed())

		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", DedupTTL: time.Hour},
			},
		}
		sender = &mocks.NotificationPublishCommandSender{}
		w := watcher.NewWatcher(cfg, []publish.Publisher{newPublisher(sender, time.Hour)})
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

	It("publishes only once for two identical events inside the window", func() {
		writeTask(tasksDir, "dup-task", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))

		writeTask(tasksDir, "dup-task", "Alice", "in_progress", "planning")
		Consistently(func() int {
			return sender.SendPublishNotificationCommandCallCount()
		}, "400ms", "20ms").Should(Equal(1))
	})
})

var _ = Describe("Watcher multi-vault", func() {
	var (
		ctx       context.Context
		cancel    context.CancelFunc
		vault1Dir string
		vault2Dir string
		tasks1Dir string
		tasks2Dir string
		sender    *mocks.NotificationPublishCommandSender
		w         watcher.Watcher
		watchDone chan error
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
				{Name: "test", DedupTTL: 5 * time.Minute},
			},
		}
		sender = &mocks.NotificationPublishCommandSender{}
		w = watcher.NewWatcher(cfg, []publish.Publisher{newPublisher(sender, 5*time.Minute)})

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

	It("publishes for tasks from vault1", func() {
		writeTask(tasks1Dir, "task-v1", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
		_, cmd := sender.SendPublishNotificationCommandArgsForCall(0)
		Expect(cmd.Metadata["taskName"]).To(Equal("task-v1"))
		Expect(cmd.Metadata["vault"]).To(Equal("vault1"))
		Expect(cmd.Message.String()).To(ContainSubstring("24%20Tasks/task-v1.md"))
	})

	It("publishes for tasks from vault2", func() {
		writeTask(tasks2Dir, "task-v2", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
		_, cmd := sender.SendPublishNotificationCommandArgsForCall(0)
		Expect(cmd.Metadata["taskName"]).To(Equal("task-v2"))
		Expect(cmd.Metadata["vault"]).To(Equal("vault2"))
		Expect(cmd.Message.String()).To(ContainSubstring("Tasks/task-v2.md"))
	})

	It("publishes for tasks from both vaults independently", func() {
		writeTask(tasks1Dir, "task-a", "Alice", "in_progress", "planning")
		writeTask(tasks2Dir, "task-b", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender.SendPublishNotificationCommandCallCount()
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

	startWatcher := func(cfg config.Config, publishers []publish.Publisher) {
		w := watcher.NewWatcher(cfg, publishers)
		watchDone = make(chan error, 1)
		go func() { watchDone <- w.Watch(ctx) }()
		time.Sleep(100 * time.Millisecond)
	}

	It("publishes through both publishers when task matches both watcher entries", func() {
		sender := &mocks.NotificationPublishCommandSender{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", DedupTTL: 5 * time.Minute},
				{Name: "w2", DedupTTL: 5 * time.Minute},
			},
		}
		startWatcher(cfg, []publish.Publisher{
			newPublisher(sender, 5*time.Minute),
			newPublisher(sender, 5*time.Minute),
		})

		writeTask(tasksDir, "both-task", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(2))
	})

	It("publishes through only the matching publisher when task matches one entry", func() {
		sender1 := &mocks.NotificationPublishCommandSender{}
		sender2 := &mocks.NotificationPublishCommandSender{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Assignee: "Alice", DedupTTL: 5 * time.Minute},
				{Name: "w2", Assignee: "Bob", DedupTTL: 5 * time.Minute},
			},
		}
		startWatcher(cfg, []publish.Publisher{
			newPublisher(sender1, 5*time.Minute),
			newPublisher(sender2, 5*time.Minute),
		})

		writeTask(tasksDir, "alice-task", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender1.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
		Consistently(func() int {
			return sender2.SendPublishNotificationCommandCallCount()
		}, "200ms", "20ms").Should(Equal(0))
	})

	It("publishes through the second publisher even when the first sender fails", func() {
		sender1 := &mocks.NotificationPublishCommandSender{}
		sender2 := &mocks.NotificationPublishCommandSender{}
		sender1.SendPublishNotificationCommandReturns(errors.New("sender 1 failed"))
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", DedupTTL: 5 * time.Minute},
				{Name: "w2", DedupTTL: 5 * time.Minute},
			},
		}
		startWatcher(cfg, []publish.Publisher{
			newPublisher(sender1, 5*time.Minute),
			newPublisher(sender2, 5*time.Minute),
		})

		writeTask(tasksDir, "error-task", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender1.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
		Eventually(func() int {
			return sender2.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
	})

	It("keeps watching and publishes later distinct events when a sender fails", func() {
		sender := &mocks.NotificationPublishCommandSender{}
		sender.SendPublishNotificationCommandReturns(errors.New("broker down"))
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", DedupTTL: 5 * time.Minute},
			},
		}
		startWatcher(cfg, []publish.Publisher{newPublisher(sender, 5*time.Minute)})

		writeTask(tasksDir, "fail-task", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))

		writeTask(tasksDir, "later-task", "Alice", "in_progress", "planning")
		Eventually(func() int {
			return sender.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(2))

		Consistently(watchDone, "200ms", "20ms").ShouldNot(Receive())
	})

	It("publishes nothing when the task assignee does not match the entry filter", func() {
		sender1 := &mocks.NotificationPublishCommandSender{}
		sender2 := &mocks.NotificationPublishCommandSender{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Assignee: "WrongUser", DedupTTL: 5 * time.Minute},
				{Name: "w2", DedupTTL: 5 * time.Minute},
			},
		}
		startWatcher(cfg, []publish.Publisher{
			newPublisher(sender1, 5*time.Minute),
			newPublisher(sender2, 5*time.Minute),
		})

		writeTask(tasksDir, "assignee-task", "Alice", "in_progress", "planning")
		Consistently(func() int {
			return sender1.SendPublishNotificationCommandCallCount()
		}, "200ms", "20ms").Should(Equal(0))
		Eventually(func() int {
			return sender2.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
	})

	It("publishes nothing when the task status does not match the entry filter", func() {
		sender1 := &mocks.NotificationPublishCommandSender{}
		sender2 := &mocks.NotificationPublishCommandSender{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Statuses: []string{"done"}, DedupTTL: 5 * time.Minute},
				{Name: "w2", DedupTTL: 5 * time.Minute},
			},
		}
		startWatcher(cfg, []publish.Publisher{
			newPublisher(sender1, 5*time.Minute),
			newPublisher(sender2, 5*time.Minute),
		})

		writeTask(tasksDir, "status-task", "Alice", "in_progress", "planning")
		Consistently(func() int {
			return sender1.SendPublishNotificationCommandCallCount()
		}, "200ms", "20ms").Should(Equal(0))
		Eventually(func() int {
			return sender2.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
	})

	It("publishes nothing when the task phase does not match the entry filter", func() {
		sender1 := &mocks.NotificationPublishCommandSender{}
		sender2 := &mocks.NotificationPublishCommandSender{}
		cfg := config.Config{
			Vaults: []config.VaultConfig{
				{Name: "v", Path: vaultDir, TasksDir: "Tasks"},
			},
			Watchers: []config.WatcherConfig{
				{Name: "w1", Phases: []string{"execution"}, DedupTTL: 5 * time.Minute},
				{Name: "w2", DedupTTL: 5 * time.Minute},
			},
		}
		startWatcher(cfg, []publish.Publisher{
			newPublisher(sender1, 5*time.Minute),
			newPublisher(sender2, 5*time.Minute),
		})

		writeTask(tasksDir, "phase-task", "Alice", "in_progress", "planning")
		Consistently(func() int {
			return sender1.SendPublishNotificationCommandCallCount()
		}, "200ms", "20ms").Should(Equal(0))
		Eventually(func() int {
			return sender2.SendPublishNotificationCommandCallCount()
		}, "500ms", "20ms").Should(Equal(1))
	})
})
