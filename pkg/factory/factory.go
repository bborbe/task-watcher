// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory

import (
	"context"

	"github.com/bborbe/cqrs/base"
	"github.com/bborbe/cqrs/cdb"
	cqrsiam "github.com/bborbe/cqrs/iam"
	"github.com/bborbe/kafka"
	"github.com/bborbe/log"
	"github.com/bborbe/notification/command/notification"
	libtime "github.com/bborbe/time"

	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/notify"
	"github.com/bborbe/task-watcher/pkg/publish"
	"github.com/bborbe/task-watcher/pkg/watcher"
)

// notificationInitiator names this producer in the command's traceability
// field. It is not an authorization: publishing is not IAM-gated.
const notificationInitiator cqrsiam.Initiator = "task-watcher"

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

// CreateNotificationSender constructs the shared-core notification command sender
// backed by the given Kafka sync producer; topicPrefix selects the command topic.
// ctx feeds the request-ID channel so downstream cancellation propagates.
// Pure composition: no network calls at construction time.
func CreateNotificationSender(
	ctx context.Context,
	syncProducer kafka.SyncProducer,
	topicPrefix base.TopicPrefix,
) notification.NotificationPublishCommandSender {
	return notification.NewNotificationPublishCommandSender(
		base.NewCommandCreator(base.RequestIDChannel(ctx)),
		cdb.NewCommandObjectSender(syncProducer, topicPrefix, log.DefaultSamplerFactory),
		notificationInitiator,
	)
}

// CreatePublishers builds one Publisher per watcher entry, in config order,
// each with that entry's dedup window.
// Pure composition: no network calls at construction time.
func CreatePublishers(
	cfg config.Config,
	sender notification.NotificationPublishCommandSender,
	topicPrefix base.TopicPrefix,
	currentDateTimeGetter libtime.CurrentDateTimeGetter,
) []publish.Publisher {
	publishers := make([]publish.Publisher, len(cfg.Watchers))
	for i, w := range cfg.Watchers {
		publishers[i] = publish.NewPublisher(sender, topicPrefix, w.DedupTTL, currentDateTimeGetter)
	}
	return publishers
}

// CreateWatcher constructs a watcher.Watcher that observes all configured vaults
// and fans out matching task events to all watcher entries.
// Pure composition: no filesystem access at construction time.
func CreateWatcher(cfg config.Config, publishers []publish.Publisher) watcher.Watcher {
	return watcher.NewWatcher(cfg, publishers)
}
