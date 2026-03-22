// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/factory"
	"github.com/bborbe/task-watcher/pkg/notify"
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

	Describe("CreateNotifiers", func() {
		It("returns empty slice when no watchers configured", func() {
			cfg := config.Config{}
			result := factory.CreateNotifiers(cfg)
			Expect(result).To(HaveLen(0))
		})

		It("returns slice of length 1 with non-nil element for openclaw-wake watcher", func() {
			cfg := config.Config{
				Watchers: []config.WatcherConfig{
					{Name: "w1", Type: "openclaw-wake", URL: "http://example.com", Token: "tok"},
				},
			}
			result := factory.CreateNotifiers(cfg)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).NotTo(BeNil())
		})

		It("returns slice of length 1 with non-nil element for telegram watcher", func() {
			cfg := config.Config{
				Watchers: []config.WatcherConfig{
					{Name: "w1", Type: "telegram", Token: "tok", ChatID: "123"},
				},
			}
			result := factory.CreateNotifiers(cfg)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).NotTo(BeNil())
		})

		It("returns slice of length 1 with non-nil element for log watcher", func() {
			cfg := config.Config{
				Watchers: []config.WatcherConfig{
					{Name: "w1", Type: "log"},
				},
			}
			result := factory.CreateNotifiers(cfg)
			Expect(result).To(HaveLen(1))
			Expect(result[0]).NotTo(BeNil())
		})
	})

	Describe("CreateWatcher", func() {
		It("returns a non-nil watcher.Watcher", func() {
			cfg := config.Config{
				Vaults: []config.VaultConfig{
					{Name: "testvault", Path: "/vault", TasksDir: "24 Tasks"},
				},
				Watchers: []config.WatcherConfig{
					{Name: "test", Type: "log"},
				},
			}
			notifiers := []notify.Notifier{factory.CreateNotifiers(cfg)[0]}
			result := factory.CreateWatcher(cfg, notifiers)
			Expect(result).NotTo(BeNil())
		})
	})
})
