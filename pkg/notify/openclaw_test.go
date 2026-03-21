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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/notify"
)

var _ = Describe("OpenClawNotifier", func() {
	var (
		ctx            context.Context
		server         *httptest.Server
		requestCount   atomic.Int32
		lastBody       []byte
		lastAuthHeader string
		statusCode     int
	)

	BeforeEach(func() {
		ctx = context.Background()
		statusCode = http.StatusOK
		requestCount.Store(0)
		lastBody = nil
		lastAuthHeader = ""

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
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

	It("sends HTTP POST with correct OpenClaw payload shape and auth header", func() {
		n := notify.NewOpenClawNotifier(server.URL, "my-secret-token", server.Client())
		notification := notify.Notification{
			TaskName: "my-task",
			Phase:    "planning",
			Assignee: "alice",
		}
		err := n.Notify(ctx, notification)
		Expect(err).NotTo(HaveOccurred())
		Expect(requestCount.Load()).To(Equal(int32(1)))
		Expect(lastAuthHeader).To(Equal("Bearer my-secret-token"))

		var payload map[string]interface{}
		Expect(json.Unmarshal(lastBody, &payload)).To(Succeed())
		Expect(
			payload["text"],
		).To(Equal("Task watcher: alice task changed (task: my-task, phase: planning)"))
		Expect(payload["mode"]).To(Equal("now"))
	})

	It("deduplicates: same task+phase only sends one request", func() {
		n := notify.NewOpenClawNotifier(server.URL, "token", server.Client())
		notification := notify.Notification{
			TaskName: "dup-task",
			Phase:    "planning",
			Assignee: "alice",
		}
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(n.Notify(ctx, notification)).To(Succeed())
		Expect(requestCount.Load()).To(Equal(int32(1)))
	})

	It("returns error when server responds with non-2xx status", func() {
		statusCode = http.StatusInternalServerError
		n := notify.NewOpenClawNotifier(server.URL, "token", server.Client())
		err := n.Notify(ctx, notify.Notification{
			TaskName: "task-a",
			Phase:    "execution",
			Assignee: "bob",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("500"))
	})
})
