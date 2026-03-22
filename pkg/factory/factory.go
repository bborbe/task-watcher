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

// CreateNotifier constructs a notify.Notifier from a validated config.
// TODO(spec-003): per-watcher notifier selection will be added in the fanout prompt.
func CreateNotifier(_ config.Config) notify.Notifier {
	return nil
}

// CreateLogNotifier constructs a notify.Notifier that logs instead of sending HTTP requests.
// TODO(spec-003): per-watcher notifier selection will be added in the fanout prompt.
func CreateLogNotifier(_ config.Config) notify.Notifier {
	return notify.NewLogNotifier(0)
}

// CreateOpenClawNotifier constructs a notify.Notifier that sends OpenClaw-formatted payloads.
// TODO(spec-003): per-watcher notifier selection will be added in the fanout prompt.
func CreateOpenClawNotifier(_ config.Config) notify.Notifier {
	return nil
}

// CreateWatcher constructs a watcher.Watcher that observes the vault and
// forwards matching task events to the notifier.
// Pure composition: no filesystem access at construction time.
func CreateWatcher(cfg config.Config, notifier notify.Notifier) watcher.Watcher {
	return watcher.NewWatcher(cfg, notifier)
}
