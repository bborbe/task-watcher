// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package publish_test

import (
	"context"
	"errors"
	stdtime "time"

	"github.com/bborbe/cqrs/base"
	libtime "github.com/bborbe/time"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mocks "github.com/bborbe/task-watcher/mocks"
	"github.com/bborbe/task-watcher/pkg/publish"
)

var baseTime = stdtime.Date(2026, stdtime.September, 13, 12, 0, 0, 0, stdtime.UTC)

var _ = Describe("Publisher", func() {
	var (
		ctx         context.Context
		sender      *mocks.NotificationPublishCommandSender
		clock       libtime.CurrentDateTime
		publisher   publish.Publisher
		topicPrefix base.TopicPrefix
		event       publish.TaskEvent
	)

	BeforeEach(func() {
		ctx = context.Background()
		sender = &mocks.NotificationPublishCommandSender{}
		clock = libtime.NewCurrentDateTime()
		clock.SetNow(libtime.DateTime(baseTime))
		topicPrefix = base.TopicPrefix("develop")
		publisher = publish.NewPublisher(sender, topicPrefix, 5*stdtime.Minute, clock)
		event = testEvent()
	})

	It("publishes one command for a matched task event", func() {
		Expect(publisher.Publish(ctx, event)).To(Succeed())

		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(1))
		_, cmd := sender.SendPublishNotificationCommandArgsForCall(0)
		Expect(string(cmd.Type)).To(Equal("agent-escalation"))
		Expect(cmd.Target).To(BeNil())
		Expect(cmd.Message).NotTo(BeEmpty())
		Expect(cmd.Metadata).To(Equal(map[string]string{
			"taskName": "My Task",
			"phase":    "human_review",
			"assignee": "Alice",
			"vault":    "testvault",
		}))
	})

	It("publishes only once for a repeated task and phase inside the window", func() {
		Expect(publisher.Publish(ctx, event)).To(Succeed())
		Expect(publisher.Publish(ctx, event)).To(Succeed())
		Expect(publisher.Publish(ctx, event)).To(Succeed())

		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(1))
	})

	It("publishes again after the injected clock passes the window", func() {
		Expect(publisher.Publish(ctx, event)).To(Succeed())
		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(1))

		clock.SetNow(libtime.DateTime(baseTime.Add(6 * stdtime.Minute)))

		Expect(publisher.Publish(ctx, event)).To(Succeed())
		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(2))
	})

	It("publishes again when the window is reached exactly", func() {
		Expect(publisher.Publish(ctx, event)).To(Succeed())

		clock.SetNow(libtime.DateTime(baseTime.Add(5 * stdtime.Minute)))

		Expect(publisher.Publish(ctx, event)).To(Succeed())
		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(2))
	})

	It("publishes a different phase for the same task", func() {
		Expect(publisher.Publish(ctx, event)).To(Succeed())

		other := event
		other.Phase = "execution"

		Expect(publisher.Publish(ctx, other)).To(Succeed())
		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(2))
	})

	It("publishes the same phase for a different task", func() {
		Expect(publisher.Publish(ctx, event)).To(Succeed())

		other := event
		other.TaskName = "Other Task"

		Expect(publisher.Publish(ctx, other)).To(Succeed())
		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(2))
	})

	It("publishes once per publisher when two publishers carry different windows", func() {
		short := publish.NewPublisher(sender, topicPrefix, stdtime.Second, clock)
		long := publish.NewPublisher(sender, topicPrefix, stdtime.Hour, clock)

		Expect(short.Publish(ctx, event)).To(Succeed())
		Expect(long.Publish(ctx, event)).To(Succeed())

		Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(2))
	})

	Context("when the sender fails", func() {
		var sendErr error

		BeforeEach(func() {
			sendErr = errors.New("broker down")
			sender.SendPublishNotificationCommandReturns(sendErr)
		})

		It("returns the wrapped error and does not retry inside the window", func() {
			err := publisher.Publish(ctx, event)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("broker down"))
			Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(1))

			Expect(publisher.Publish(ctx, event)).To(Succeed())
			Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(1))
		})

		It("publishes again after the window passes", func() {
			Expect(publisher.Publish(ctx, event)).To(HaveOccurred())

			clock.SetNow(libtime.DateTime(baseTime.Add(6 * stdtime.Minute)))

			Expect(publisher.Publish(ctx, event)).To(HaveOccurred())
			Expect(sender.SendPublishNotificationCommandCallCount()).To(Equal(2))
		})
	})

	It("publishes a validated command carrying the topic prefix", func() {
		Expect(publisher.Publish(ctx, event)).To(Succeed())

		_, cmd := sender.SendPublishNotificationCommandArgsForCall(0)
		Expect(cmd.Validate(ctx)).To(Succeed())
		Expect(cmd.Target).To(BeNil())
	})
})
