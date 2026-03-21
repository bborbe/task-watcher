// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/mocks"
	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/factory"
)

var _ = Describe("Factory", func() {
	Describe("CreateConfigLoader", func() {
		It("returns a non-nil config.Loader", func() {
			result := factory.CreateConfigLoader("/some/path")
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("CreateNotifier", func() {
		It("returns a non-nil notify.Notifier", func() {
			cfg := config.Config{Webhook: "http://example.com"}
			result := factory.CreateNotifier(cfg)
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("CreateDryRunNotifier", func() {
		It("returns a non-nil notify.Notifier", func() {
			cfg := config.Config{Webhook: "http://example.com"}
			result := factory.CreateDryRunNotifier(cfg)
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("CreateWatcher", func() {
		It("returns a non-nil watcher.Watcher", func() {
			cfg := config.Config{
				Vaults: []config.VaultConfig{
					{Name: "testvault", Path: "/vault", TasksDir: "24 Tasks"},
				},
				Assignee: "user",
				Statuses: []string{"active"},
				Phases:   []string{"phase1"},
				Webhook:  "http://example.com",
			}
			notifier := &mocks.FakeNotifier{}
			result := factory.CreateWatcher(cfg, notifier)
			Expect(result).NotTo(BeNil())
		})
	})
})
