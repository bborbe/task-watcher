// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory

import (
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
//
// CreateNotifiers is an interim placeholder: watcher entries are filter-only, so the
// bespoke path can no longer be selected per type. Delivery is log-only until the shared
// publish path lands in the next prompt; this constructor and pkg/notify are removed by
// the gated deletion prompt of spec 004.
func CreateNotifiers(cfg config.Config) []notify.Notifier {
	notifiers := make([]notify.Notifier, len(cfg.Watchers))
	for i, w := range cfg.Watchers {
		notifiers[i] = notify.NewLogNotifier(w.DedupTTL)
	}
	return notifiers
}

// CreateWatcher constructs a watcher.Watcher that observes all configured vaults
// and fans out matching task events to all watcher entries.
// Pure composition: no filesystem access at construction time.
func CreateWatcher(cfg config.Config, notifiers []notify.Notifier) watcher.Watcher {
	return watcher.NewWatcher(cfg, notifiers)
}
