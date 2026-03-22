// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/notify"
)

var _ = Describe("LogNotifier", func() {
	var (
		ctx    context.Context
		buf    bytes.Buffer
		logger *slog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		buf.Reset()
		logger = slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		slog.SetDefault(logger)
	})

	It("returns nil on first notification", func() {
		n := notify.NewLogNotifier(time.Minute)
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "planning",
			Assignee: "alice",
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("logs the notification event", func() {
		n := notify.NewLogNotifier(time.Minute)
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "planning",
			Assignee: "alice",
		})).To(Succeed())
		output := buf.String()
		Expect(output).To(ContainSubstring("log notifier: task event"))
		Expect(output).To(ContainSubstring("task=task-a"))
		Expect(output).To(ContainSubstring("phase=planning"))
		Expect(output).To(ContainSubstring("assignee=alice"))
	})

	It("deduplicates: second call within TTL produces no additional log line", func() {
		n := notify.NewLogNotifier(time.Minute)
		notification := notify.Notification{
			TaskName: "task-dup",
			Phase:    "execution",
			Assignee: "bob",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		firstOutput := buf.String()
		countBefore := strings.Count(firstOutput, "log notifier: task event")

		Expect(n.Notify(ctx, notification)).To(Succeed())
		countAfter := strings.Count(buf.String(), "log notifier: task event")

		Expect(countAfter).To(Equal(countBefore)) // no new log line
	})

	It("re-sends after TTL expires", func() {
		n := notify.NewLogNotifier(50 * time.Millisecond)
		notification := notify.Notification{
			TaskName: "task-ttl",
			Phase:    "planning",
			Assignee: "alice",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		countAfterFirst := strings.Count(buf.String(), "log notifier: task event")

		// Within TTL — should be deduped
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(strings.Count(buf.String(), "log notifier: task event")).To(Equal(countAfterFirst))

		// Wait for TTL to expire
		time.Sleep(60 * time.Millisecond)

		// After TTL — should log again
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(
			strings.Count(buf.String(), "log notifier: task event"),
		).To(Equal(countAfterFirst + 1))
	})

	It("does not deduplicate different task names", func() {
		n := notify.NewLogNotifier(time.Minute)
		Expect(
			n.Notify(
				ctx,
				notify.Notification{TaskName: "task-a", Phase: "planning", Assignee: "alice"},
			),
		).To(Succeed())
		Expect(
			n.Notify(
				ctx,
				notify.Notification{TaskName: "task-b", Phase: "planning", Assignee: "alice"},
			),
		).To(Succeed())
		Expect(strings.Count(buf.String(), "log notifier: task event")).To(Equal(2))
	})

	It("does not deduplicate different phases for same task", func() {
		n := notify.NewLogNotifier(time.Minute)
		Expect(
			n.Notify(
				ctx,
				notify.Notification{TaskName: "task-a", Phase: "planning", Assignee: "alice"},
			),
		).To(Succeed())
		Expect(
			n.Notify(
				ctx,
				notify.Notification{TaskName: "task-a", Phase: "execution", Assignee: "alice"},
			),
		).To(Succeed())
		Expect(strings.Count(buf.String(), "log notifier: task event")).To(Equal(2))
	})
})
