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

var _ = Describe("Notifier", func() {
	var (
		ctx          context.Context
		server       *httptest.Server
		requestCount atomic.Int32
		lastBody     []byte
		statusCode   int
	)

	BeforeEach(func() {
		ctx = context.Background()
		statusCode = http.StatusOK
		requestCount.Store(0)
		lastBody = nil

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
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

	It("sends HTTP POST with correct JSON body on success", func() {
		n := notify.NewNotifier(server.URL, server.Client(), time.Minute)
		notification := notify.Notification{
			TaskName: "my-task",
			Phase:    "planning",
			Assignee: "alice",
		}
		err := n.Notify(ctx, notification)
		Expect(err).NotTo(HaveOccurred())
		Expect(requestCount.Load()).To(Equal(int32(1)))

		var got notify.Notification
		Expect(json.Unmarshal(lastBody, &got)).To(Succeed())
		Expect(got.TaskName).To(Equal("my-task"))
		Expect(got.Phase).To(Equal("planning"))
		Expect(got.Assignee).To(Equal("alice"))
	})

	It("returns error when server responds with 500", func() {
		statusCode = http.StatusInternalServerError
		n := notify.NewNotifier(server.URL, server.Client(), time.Minute)
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "execution",
			Assignee: "bob",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("500"))
	})

	It("returns error when server responds with 404", func() {
		statusCode = http.StatusNotFound
		n := notify.NewNotifier(server.URL, server.Client(), time.Minute)
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-b",
			Phase:    "review",
			Assignee: "carol",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("404"))
	})

	It("deduplicates: same task+phase only sends one request", func() {
		n := notify.NewNotifier(server.URL, server.Client(), time.Minute)
		notification := notify.Notification{
			TaskName: "dup-task",
			Phase:    "planning",
			Assignee: "alice",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(requestCount.Load()).To(Equal(int32(1)))
	})

	It("does not deduplicate different phases for same task", func() {
		n := notify.NewNotifier(server.URL, server.Client(), time.Minute)
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
		Expect(requestCount.Load()).To(Equal(int32(2)))
	})

	It("does not deduplicate different tasks with same phase", func() {
		n := notify.NewNotifier(server.URL, server.Client(), time.Minute)
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-1",
			Phase:    "planning",
			Assignee: "alice",
		})).To(Succeed())
		Expect(n.Notify(ctx, notify.Notification{
			TaskName: "task-2",
			Phase:    "planning",
			Assignee: "alice",
		})).To(Succeed())
		Expect(requestCount.Load()).To(Equal(int32(2)))
	})

	It("re-sends webhook after TTL expires", func() {
		n := notify.NewNotifier(server.URL, server.Client(), 50*time.Millisecond)
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
})
