// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/notify"
)

var _ = Describe("DryRunOpenClawNotifier", func() {
	var (
		ctx context.Context
		n   notify.Notifier
	)

	BeforeEach(func() {
		ctx = context.Background()
		n = notify.NewDryRunOpenClawNotifier(
			"https://example.com/hooks/agent",
			"my-token",
			time.Minute,
		)
	})

	It("returns nil on first notification", func() {
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "planning",
			Assignee: "alice",
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("deduplicates: second call with same task+phase returns nil", func() {
		notification := notify.Notification{
			TaskName: "task-dup",
			Phase:    "execution",
			Assignee: "bob",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(n.Notify(ctx, notification)).To(Succeed())
	})

	It("does not deduplicate different task+phase combinations", func() {
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-x",
			Phase:    "planning",
			Assignee: "alice",
		})).To(Succeed())
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-x",
			Phase:    "execution",
			Assignee: "alice",
		})).To(Succeed())
	})

	It("re-sends after TTL expires", func() {
		shortTTL := notify.NewDryRunOpenClawNotifier(
			"https://example.com/hooks/agent",
			"token",
			50*time.Millisecond,
		)
		notification := notify.Notification{
			TaskName: "task-ttl",
			Phase:    "planning",
			Assignee: "alice",
		}
		Expect(shortTTL.Notify(ctx, notification)).To(Succeed())
		Expect(shortTTL.Notify(ctx, notification)).To(Succeed()) // within TTL, deduped
		time.Sleep(60 * time.Millisecond)
		Expect(shortTTL.Notify(ctx, notification)).To(Succeed()) // after TTL, re-sends
	})
})

var _ = Describe("DryRunNotifier", func() {
	var (
		ctx context.Context
		n   notify.Notifier
	)

	BeforeEach(func() {
		ctx = context.Background()
		n = notify.NewDryRunNotifier("https://example.com/webhook", time.Minute)
	})

	It("returns nil on first notification", func() {
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "planning",
			Assignee: "alice",
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("deduplicates: second call with same task+phase returns nil without re-logging", func() {
		notification := notify.Notification{
			TaskName: "task-dup",
			Phase:    "execution",
			Assignee: "bob",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(n.Notify(ctx, notification)).To(Succeed())
	})

	It("does not deduplicate different task+phase combinations", func() {
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-x",
			Phase:    "planning",
			Assignee: "alice",
		})).To(Succeed())
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-x",
			Phase:    "execution",
			Assignee: "alice",
		})).To(Succeed())
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-y",
			Phase:    "planning",
			Assignee: "alice",
		})).To(Succeed())
	})

	It("re-sends after TTL expires", func() {
		shortTTL := notify.NewDryRunNotifier("https://example.com/webhook", 50*time.Millisecond)
		notification := notify.Notification{
			TaskName: "task-ttl",
			Phase:    "planning",
			Assignee: "alice",
		}
		Expect(shortTTL.Notify(ctx, notification)).To(Succeed())
		Expect(shortTTL.Notify(ctx, notification)).To(Succeed()) // within TTL, deduped
		time.Sleep(60 * time.Millisecond)
		Expect(shortTTL.Notify(ctx, notification)).To(Succeed()) // after TTL, re-sends
	})
})
