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

// CreateNotifiers builds one Notifier per WatcherConfig entry in the order they appear.
// Pure composition: no network calls at construction time.
func CreateNotifiers(cfg config.Config) []notify.Notifier {
	notifiers := make([]notify.Notifier, len(cfg.Watchers))
	for i, w := range cfg.Watchers {
		notifiers[i] = createNotifierForWatcher(w)
	}
	return notifiers
}

// CreateWatcher constructs a watcher.Watcher that observes all configured vaults
// and fans out matching task events to all watcher entries.
// Pure composition: no filesystem access at construction time.
func CreateWatcher(cfg config.Config, notifiers []notify.Notifier) watcher.Watcher {
	return watcher.NewWatcher(cfg, notifiers)
}

// createNotifierForWatcher instantiates the correct Notifier implementation based on watcher type.
func createNotifierForWatcher(w config.WatcherConfig) notify.Notifier {
	switch w.Type {
	case "openclaw-wake":
		return notify.NewOpenClawNotifier(w.URL, w.Token, http.DefaultClient, w.DedupTTL)
	case "telegram":
		return notify.NewTelegramNotifier(w.Token, w.ChatID, http.DefaultClient, w.DedupTTL)
	case "log":
		return notify.NewLogNotifier(w.DedupTTL)
	default:
		// Should never reach here — validated in config.Load
		panic("unknown watcher type: " + w.Type)
	}
}
