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

// openClawPayload is the JSON payload sent to the OpenClaw /hooks/agent endpoint.
type openClawPayload struct {
	Name     string `json:"name"`
	Message  string `json:"message"`
	WakeMode string `json:"wakeMode"`
	Deliver  bool   `json:"deliver"`
}

// NewOpenClawNotifier returns a Notifier that posts to an OpenClaw /hooks/agent endpoint.
func NewOpenClawNotifier(webhookURL string, token string, httpClient *http.Client) Notifier {
	return &openClawNotifier{
		webhookURL: webhookURL,
		token:      token,
		httpClient: httpClient,
		seen:       make(map[string]struct{}),
	}
}

type openClawNotifier struct {
	webhookURL string
	token      string
	httpClient *http.Client
	seen       map[string]struct{}
	mu         sync.Mutex
}

func (n *openClawNotifier) Notify(ctx context.Context, notification Notification) error {
	key := notification.TaskName + ":" + notification.Phase

	n.mu.Lock()
	_, exists := n.seen[key]
	if exists {
		n.mu.Unlock()
		return nil
	}
	n.seen[key] = struct{}{}
	n.mu.Unlock()

	payload := openClawPayload{
		Name: "task-watcher",
		Message: fmt.Sprintf(
			"Task update: %s. Assignee: %s. Phase: %s.",
			notification.TaskName,
			notification.Assignee,
			notification.Phase,
		),
		WakeMode: "now",
		Deliver:  false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(ctx, err, "marshal openclaw payload")
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
	req.Header.Set("Authorization", "Bearer "+n.token)

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
