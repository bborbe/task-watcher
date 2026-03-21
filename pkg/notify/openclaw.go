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
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/bborbe/errors"
)

// openClawPayload is the JSON payload sent to the OpenClaw /hooks/wake endpoint.
type openClawPayload struct {
	Text string `json:"text"`
	Mode string `json:"mode"`
}

// NewOpenClawNotifier returns a Notifier that posts to an OpenClaw /hooks/wake endpoint.
func NewOpenClawNotifier(
	webhookURL string,
	token string,
	httpClient *http.Client,
	dedupTTL time.Duration,
) Notifier {
	return &openClawNotifier{
		webhookURL: webhookURL,
		token:      token,
		httpClient: httpClient,
		dedupTTL:   dedupTTL,
		seen:       make(map[string]time.Time),
	}
}

type openClawNotifier struct {
	webhookURL string
	token      string
	httpClient *http.Client
	dedupTTL   time.Duration
	seen       map[string]time.Time
	mu         sync.Mutex
}

func (n *openClawNotifier) Notify(ctx context.Context, notification Notification) error {
	key := notification.TaskName + ":" + notification.Phase

	n.mu.Lock()
	lastSent, exists := n.seen[key]
	if exists && time.Since(lastSent) < n.dedupTTL {
		n.mu.Unlock()
		slog.Debug("webhook skipped (duplicate within TTL)",
			"task", notification.TaskName,
			"phase", notification.Phase,
			"ttl", n.dedupTTL,
			"lastSent", lastSent,
		)
		return nil
	}
	n.seen[key] = time.Now()
	n.mu.Unlock()

	payload := openClawPayload{
		Text: fmt.Sprintf(
			"Task watcher: %s task changed (task: %s, phase: %s)",
			notification.Assignee,
			notification.TaskName,
			notification.Phase,
		),
		Mode: "now",
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

	slog.Debug("sending webhook",
		"method", http.MethodPost,
		"url", n.webhookURL,
		"header", "Content-Type: application/json",
		"header", "Authorization: Bearer ***",
		"body", string(body),
	)

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

	slog.Info(
		"webhook sent",
		"task",
		notification.TaskName,
		"phase",
		notification.Phase,
		"status",
		resp.StatusCode,
	)
	return nil
}
