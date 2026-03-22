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

// telegramPayload is the JSON payload sent to the Telegram Bot API sendMessage endpoint.
type telegramPayload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// NewTelegramNotifier returns a Notifier that sends messages via the Telegram Bot API.
// The bot token is never logged.
func NewTelegramNotifier(
	token string,
	chatID string,
	httpClient *http.Client,
	dedupTTL time.Duration,
) Notifier {
	return NewTelegramNotifierWithBaseURL(
		token,
		chatID,
		httpClient,
		dedupTTL,
		"https://api.telegram.org",
	)
}

// NewTelegramNotifierWithBaseURL returns a Notifier that sends messages via the Telegram Bot API
// using a custom base URL. Intended for testing only.
func NewTelegramNotifierWithBaseURL(
	token string,
	chatID string,
	httpClient *http.Client,
	dedupTTL time.Duration,
	baseURL string,
) Notifier {
	return &telegramNotifier{
		token:      token,
		chatID:     chatID,
		httpClient: httpClient,
		dedupTTL:   dedupTTL,
		baseURL:    baseURL,
		seen:       make(map[string]time.Time),
	}
}

type telegramNotifier struct {
	token      string
	chatID     string
	httpClient *http.Client
	dedupTTL   time.Duration
	baseURL    string
	seen       map[string]time.Time
	mu         sync.Mutex
}

func (t *telegramNotifier) Notify(ctx context.Context, notification Notification) error {
	key := notification.TaskName + ":" + notification.Phase

	t.mu.Lock()
	lastSent, exists := t.seen[key]
	if exists && time.Since(lastSent) < t.dedupTTL {
		t.mu.Unlock()
		slog.Debug("telegram skipped (duplicate within TTL)",
			"task", notification.TaskName,
			"phase", notification.Phase,
			"ttl", t.dedupTTL,
			"lastSent", lastSent,
		)
		return nil
	}
	t.seen[key] = time.Now()
	t.mu.Unlock()

	payload := telegramPayload{
		ChatID: t.chatID,
		Text: fmt.Sprintf(
			"Task watcher: %s task changed (task: %s, phase: %s)",
			notification.Assignee,
			notification.TaskName,
			notification.Phase,
		),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(ctx, err, "marshal telegram payload")
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return errors.Wrapf(ctx, err, "create http request")
	}
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("sending telegram message",
		"method", http.MethodPost,
		"chat_id", t.chatID,
		"body", string(body),
	)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(ctx, err, "execute http request")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.Errorf(
			ctx,
			"telegram returned non-2xx status: %s",
			fmt.Sprintf("%d", resp.StatusCode),
		)
	}

	slog.Info("telegram message sent",
		"task", notification.TaskName,
		"phase", notification.Phase,
		"status", resp.StatusCode,
	)
	return nil
}
