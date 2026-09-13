// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package publish_test

import (
	"context"

	"github.com/bborbe/cqrs/base"
	core "github.com/bborbe/notification"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/publish"
)

func testEvent() publish.TaskEvent {
	return publish.TaskEvent{
		Vault:    "testvault",
		TasksDir: "24 Tasks",
		TaskName: "My Task",
		Status:   "in_progress",
		Phase:    "human_review",
		Assignee: "Alice",
	}
}

var _ = Describe("BuildCommand", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("builds the publish command the shared core expects", func() {
		cmd, err := publish.BuildCommand(ctx, testEvent())
		Expect(err).NotTo(HaveOccurred())

		Expect(string(cmd.Type)).To(Equal("agent-escalation"))
		Expect(cmd.Type).To(Equal(core.AgentEscalationNotificationType))
		Expect(cmd.Target).To(BeNil())
		Expect(cmd.Message).NotTo(BeEmpty())
		Expect(cmd.Metadata).To(Equal(map[string]string{
			"taskName": "My Task",
			"phase":    "human_review",
			"assignee": "Alice",
			"vault":    "testvault",
		}))
	})

	It("puts the task, status, phase, assignee and deeplink in the message", func() {
		cmd, err := publish.BuildCommand(ctx, testEvent())
		Expect(err).NotTo(HaveOccurred())

		Expect(cmd.Message.String()).To(ContainSubstring("My Task"))
		Expect(cmd.Message.String()).To(ContainSubstring("in_progress"))
		Expect(cmd.Message.String()).To(ContainSubstring("human_review"))
		Expect(cmd.Message.String()).To(ContainSubstring("Alice"))
		Expect(cmd.Message.String()).To(ContainSubstring(
			"obsidian://open?vault=testvault&file=24%20Tasks/My%20Task.md",
		))
	})

	It("percent-encodes spaces as %20 and never as +, keeping the slash", func() {
		cmd, err := publish.BuildCommand(ctx, testEvent())
		Expect(err).NotTo(HaveOccurred())

		Expect(cmd.Message.String()).To(ContainSubstring("24%20Tasks/My%20Task.md"))
		Expect(cmd.Message.String()).NotTo(ContainSubstring("24+Tasks"))
		Expect(cmd.Message.String()).NotTo(ContainSubstring("My+Task"))
	})

	It("builds a command that passes the library's own validation", func() {
		cmd, err := publish.BuildCommand(ctx, testEvent())
		Expect(err).NotTo(HaveOccurred())

		Expect(cmd.Type.Validate(ctx)).To(Succeed())
		Expect(cmd.Validate(ctx)).To(Succeed())
	})

	It("survives the serialisation the library performs on the wire", func() {
		cmd, err := publish.BuildCommand(ctx, testEvent())
		Expect(err).NotTo(HaveOccurred())

		event, err := base.ParseEvent(ctx, cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(event["type"]).To(Equal("agent-escalation"))
		_, hasTarget := event["target"]
		Expect(hasTarget).To(BeFalse())
	})

	It("returns an error when the event carries no vault", func() {
		event := testEvent()
		event.Vault = ""
		_, err := publish.BuildCommand(ctx, event)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when the event carries no task name", func() {
		event := testEvent()
		event.TaskName = ""
		_, err := publish.BuildCommand(ctx, event)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("BuildCommand command type", func() {
	It("is the registered agent-escalation type, not a hand-written literal", func() {
		cmd, err := publish.BuildCommand(context.Background(), testEvent())
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd.Type).To(Equal(core.AgentEscalationNotificationType))
		Expect(
			core.AvailableNotificationTypes.Contains(cmd.Type),
		).To(BeTrue(), "the type must be on the library's allowlist")
		Expect(cmd.Target).To(BeNil())
	})
})
