// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/bborbe/errors"
)

// Notification holds the data sent to the webhook.
type Notification struct {
	TaskName string `json:"task_name"`
	Phase    string `json:"phase"`
	Assignee string `json:"assignee"`
}

// Notifier sends notifications to a configured webhook URL.
//
//counterfeiter:generate -o ../../mocks/notifier.go --fake-name FakeNotifier . Notifier
type Notifier interface {
	Notify(ctx context.Context, notification Notification) error
}

// NewNotifier returns a Notifier that posts JSON to webhookURL.
func NewNotifier(webhookURL string, httpClient *http.Client) Notifier {
	return &notifier{
		webhookURL: webhookURL,
		httpClient: httpClient,
		seen:       make(map[string]struct{}),
	}
}

type notifier struct {
	webhookURL string
	httpClient *http.Client
	seen       map[string]struct{}
	mu         sync.Mutex
}

func (n *notifier) Notify(ctx context.Context, notification Notification) error {
	key := notification.TaskName + ":" + notification.Phase

	n.mu.Lock()
	_, exists := n.seen[key]
	if exists {
		n.mu.Unlock()
		return nil
	}
	n.seen[key] = struct{}{}
	n.mu.Unlock()

	body, err := json.Marshal(notification)
	if err != nil {
		return errors.Wrapf(ctx, err, "marshal notification")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		n.webhookURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return errors.Wrapf(ctx, err, "create http request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(ctx, err, "execute http request")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.Errorf(
			ctx,
			"webhook returned non-2xx status: %s",
			fmt.Sprintf("%d", resp.StatusCode),
		)
	}

	return nil
}
