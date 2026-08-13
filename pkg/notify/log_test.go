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

	libtime "github.com/bborbe/time"
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

var _ = Describe("LogNotifier dedup clock", func() {
	var originalNow func() time.Time
	var ctx context.Context

	BeforeEach(func() { originalNow = libtime.Now; ctx = context.Background() })
	AfterEach(func() { libtime.Now = originalNow })

	// Regression: the dedup write and the TTL read must use the SAME clock.
	// Before this fix the entry was stamped with libtime.Now() but expiry was
	// measured with time.Since(), so advancing a fake clock past the TTL did
	// not expire the entry and the second notify was still suppressed.
	It("expires a dedup entry when the injected clock passes the TTL", func() {
		current := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
		libtime.Now = func() time.Time { return current }

		var buf bytes.Buffer
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

		notifier := notify.NewLogNotifier(50 * time.Millisecond)
		n := notify.Notification{TaskName: "t", Phase: "p"}

		Expect(notifier.Notify(ctx, n)).To(Succeed())
		first := strings.Count(buf.String(), "log notifier: task event")

		// Same instant: still inside the TTL, so this one is deduped.
		Expect(notifier.Notify(ctx, n)).To(Succeed())
		Expect(strings.Count(buf.String(), "log notifier: task event")).To(Equal(first))

		// Advance the injected clock past the TTL -- no real sleeping.
		current = current.Add(time.Second)
		Expect(notifier.Notify(ctx, n)).To(Succeed())
		Expect(
			strings.Count(buf.String(), "log notifier: task event"),
		).To(BeNumerically(">", first))
	})
})
