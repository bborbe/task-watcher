// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory

import (
	"net/http"

	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/notify"
	"github.com/bborbe/task-watcher/pkg/watcher"
)

// CreateConfigLoader constructs a config.Loader for the given file path.
// Pure composition: no I/O, no context creation.
func CreateConfigLoader(filePath string) config.Loader {
	return config.NewLoader(filePath)
}

// CreateNotifier constructs a notify.Notifier from a validated config.
// Pure composition: no network calls at construction time.
func CreateNotifier(cfg config.Config) notify.Notifier {
	return notify.NewNotifier(cfg.Webhook, http.DefaultClient, cfg.DedupTTL)
}

// CreateDryRunNotifier constructs a notify.Notifier that logs instead of sending HTTP requests.
// Pure composition: no I/O at construction time.
func CreateDryRunNotifier(cfg config.Config) notify.Notifier {
	return notify.NewDryRunNotifier(cfg.Webhook, cfg.DedupTTL)
}

// CreateOpenClawNotifier constructs a notify.Notifier that sends OpenClaw-formatted payloads.
// Pure composition: no network calls at construction time.
func CreateOpenClawNotifier(cfg config.Config) notify.Notifier {
	return notify.NewOpenClawNotifier(
		cfg.Webhook,
		cfg.WebhookToken,
		http.DefaultClient,
		cfg.DedupTTL,
	)
}

// CreateDryRunOpenClawNotifier constructs a notify.Notifier that logs OpenClaw-formatted payloads instead of sending HTTP requests.
// Pure composition: no I/O at construction time.
func CreateDryRunOpenClawNotifier(cfg config.Config) notify.Notifier {
	return notify.NewDryRunOpenClawNotifier(cfg.Webhook, cfg.WebhookToken, cfg.DedupTTL)
}

// CreateWatcher constructs a watcher.Watcher that observes the vault and
// forwards matching task events to the notifier.
// Pure composition: no filesystem access at construction time.
func CreateWatcher(cfg config.Config, notifier notify.Notifier) watcher.Watcher {
	return watcher.NewWatcher(cfg, notifier)
}
