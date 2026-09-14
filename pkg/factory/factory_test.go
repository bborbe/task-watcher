// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory_test

import (
	"time"

	"github.com/bborbe/cqrs/base"
	libtime "github.com/bborbe/time"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mocks "github.com/bborbe/task-watcher/mocks"
	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/factory"
)

var _ = Describe("Factory", func() {
	Describe("CreateConfigLoader", func() {
		It("returns a non-nil config.Loader", func() {
			result := factory.CreateConfigLoader("/some/path")
			Expect(result).NotTo(BeNil())
		})

		It("returns a non-nil config.Loader for empty path", func() {
			result := factory.CreateConfigLoader("")
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("CreatePublishers", func() {
		var sender *mocks.NotificationPublishCommandSender

		BeforeEach(func() {
			sender = &mocks.NotificationPublishCommandSender{}
		})

		It("returns empty slice when no watchers configured", func() {
			result := factory.CreatePublishers(
				config.Config{},
				sender,
				base.TopicPrefix("develop"),
				libtime.NewCurrentDateTime(),
			)
			Expect(result).To(HaveLen(0))
		})

		It("returns one non-nil publisher per watcher entry, in order", func() {
			cfg := config.Config{
				Watchers: []config.WatcherConfig{
					{Name: "w1", Assignee: "alice", DedupTTL: 5 * time.Minute},
					{Name: "w2", Statuses: []string{"in_progress"}, DedupTTL: time.Minute},
					{Name: "w3"},
				},
			}
			result := factory.CreatePublishers(
				cfg,
				sender,
				base.TopicPrefix("develop"),
				libtime.NewCurrentDateTime(),
			)
			Expect(result).To(HaveLen(3))
			for _, p := range result {
				Expect(p).NotTo(BeNil())
			}
		})
	})

	Describe("CreateWatcher", func() {
		It("returns a non-nil watcher.Watcher", func() {
			cfg := config.Config{
				Vaults: []config.VaultConfig{
					{Name: "testvault", Path: "/vault", TasksDir: "24 Tasks"},
				},
				Watchers: []config.WatcherConfig{
					{Name: "test", DedupTTL: 5 * time.Minute},
				},
			}
			publishers := factory.CreatePublishers(
				cfg,
				&mocks.NotificationPublishCommandSender{},
				base.TopicPrefix("develop"),
				libtime.NewCurrentDateTime(),
			)
			result := factory.CreateWatcher(cfg, publishers)
			Expect(result).NotTo(BeNil())
		})
	})
})
