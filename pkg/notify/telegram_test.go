// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/notify"
)

var _ = Describe("TelegramNotifier", func() {
	var (
		ctx            context.Context
		server         *httptest.Server
		requestCount   atomic.Int32
		lastBody       []byte
		lastPath       string
		lastAuthHeader string
		statusCode     int
	)

	BeforeEach(func() {
		ctx = context.Background()
		statusCode = http.StatusOK
		requestCount.Store(0)
		lastBody = nil
		lastPath = ""
		lastAuthHeader = ""

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			lastPath = r.URL.Path
			lastAuthHeader = r.Header.Get("Authorization")
			var err error
			lastBody, err = io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(statusCode)
		}))
		DeferCleanup(server.Close)
	})

	It("sends POST to correct Telegram URL path /bot<token>/sendMessage", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"mytoken",
			"123456",
			server.Client(),
			time.Minute,
			server.URL,
		)
		err := n.Notify(ctx, notify.Notification{
			TaskName: "my-task",
			Phase:    "planning",
			Assignee: "alice",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(requestCount.Load()).To(Equal(int32(1)))
		Expect(lastPath).To(Equal("/botmytoken/sendMessage"))
	})

	It("sends correct chat_id and text in body", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			time.Minute,
			server.URL,
		)
		err := n.Notify(ctx, notify.Notification{
			TaskName: "my-task",
			Phase:    "planning",
			Assignee: "alice",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(requestCount.Load()).To(Equal(int32(1)))

		var payload map[string]interface{}
		Expect(json.Unmarshal(lastBody, &payload)).To(Succeed())
		Expect(payload["chat_id"]).To(Equal("chat99"))
		Expect(
			payload["text"],
		).To(Equal("Task watcher: alice task changed (task: my-task, phase: planning)"))
	})

	It("does not set Authorization header (token is in URL)", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			time.Minute,
			server.URL,
		)
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "planning",
			Assignee: "alice",
		})).To(Succeed())
		Expect(lastAuthHeader).To(BeEmpty())
	})

	It("returns nil on success (2xx response)", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			time.Minute,
			server.URL,
		)
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "planning",
			Assignee: "alice",
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error on non-2xx response", func() {
		statusCode = http.StatusBadRequest
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			time.Minute,
			server.URL,
		)
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "execution",
			Assignee: "bob",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("400"))
	})

	It("deduplicates: same task+phase only sends one request within TTL", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			time.Minute,
			server.URL,
		)
		notification := notify.Notification{
			TaskName: "dup-task",
			Phase:    "planning",
			Assignee: "alice",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(requestCount.Load()).To(Equal(int32(1)))
	})

	It("re-sends after TTL expires", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			50*time.Millisecond,
			server.URL,
		)
		notification := notify.Notification{
			TaskName: "retry-task",
			Phase:    "planning",
			Assignee: "alice",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(requestCount.Load()).To(Equal(int32(1)))

		// Within TTL — should be deduped
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(requestCount.Load()).To(Equal(int32(1)))

		// Wait for TTL to expire
		time.Sleep(60 * time.Millisecond)

		// After TTL — should re-send
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(requestCount.Load()).To(Equal(int32(2)))
	})

	It("does not deduplicate different task names", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			time.Minute,
			server.URL,
		)
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
		Expect(requestCount.Load()).To(Equal(int32(2)))
	})

	It("does not deduplicate different phases for same task", func() {
		n := notify.NewTelegramNotifierWithBaseURL(
			"tok",
			"chat99",
			server.Client(),
			time.Minute,
			server.URL,
		)
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
		Expect(requestCount.Load()).To(Equal(int32(2)))
	})
})
