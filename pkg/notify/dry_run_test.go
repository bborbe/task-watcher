// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/notify"
)

var _ = Describe("DryRunNotifier", func() {
	var (
		ctx context.Context
		n   notify.Notifier
	)

	BeforeEach(func() {
		ctx = context.Background()
		n = notify.NewDryRunNotifier("https://example.com/webhook")
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
})
