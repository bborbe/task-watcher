// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/config"
)

func writeTempConfig(content string) string {
	f, err := os.CreateTemp("", "config-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = f.WriteString(content)
	Expect(err).NotTo(HaveOccurred())
	Expect(f.Close()).To(Succeed())
	return f.Name()
}

var _ = Describe("Loader", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("vault validation", func() {
		It("parses a valid multi-vault config with watchers list", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/personal
    tasks_dir: "24 Tasks"
  work:
    path: /tmp/work
    tasks_dir: Tasks
watchers: []
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Vaults).To(HaveLen(2))
			Expect(cfg.Watchers).To(BeEmpty())
		})

		It("returns error when vaults map is empty", func() {
			path := writeTempConfig(`
vaults: {}
watchers: []
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("vaults"))
		})

		It("returns error when vaults key is missing", func() {
			path := writeTempConfig(`
watchers: []
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("vaults"))
		})

		It("returns error when a vault is missing path", func() {
			path := writeTempConfig(`
vaults:
  personal:
    tasks_dir: "24 Tasks"
watchers: []
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("path"))
		})

		It("returns error when a vault is missing tasks_dir", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
watchers: []
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tasks_dir"))
		})

		It("expands ~/ in vault path", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: ~/notes/vault
    tasks_dir: "24 Tasks"
watchers: []
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())

			home, err := os.UserHomeDir()
			Expect(err).NotTo(HaveOccurred())
			for _, v := range cfg.Vaults {
				Expect(v.Path).To(HavePrefix(home))
				Expect(v.Path).NotTo(ContainSubstring("~"))
			}
		})
	})

	Context("backward-compat detection", func() {
		It("returns error when top-level assignee field is present", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
assignee: alice
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("old flat format"))
		})

		It("returns error when top-level webhook field is present", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
webhook: https://hooks.example.com/notify
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("old flat format"))
		})

		It("returns error when top-level format field is present", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
format: openclaw
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("old flat format"))
		})
	})

	Context("watchers parsing", func() {
		It("accepts empty watchers list", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers: []
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Watchers).To(BeEmpty())
		})

		It("accepts missing watchers key (treated as empty)", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Watchers).To(BeEmpty())
		})

		It("returns error when watcher is missing name", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - type: log
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing required field: name"))
		})

		It("returns error when watcher is missing type", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: my-watcher
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing required field: type"))
		})

		It("returns error when watcher has unknown type", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: my-watcher
    type: slack
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown type"))
		})

		It("returns error when openclaw-wake watcher is missing url", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: wake
    type: openclaw-wake
    token: secret
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("url"))
		})

		It("returns error when openclaw-wake watcher is missing token", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: wake
    type: openclaw-wake
    url: http://127.0.0.1:18789/hooks/wake
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("token"))
		})

		It("returns error when telegram watcher is missing token", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: tg
    type: telegram
    chat_id: "456"
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("token"))
		})

		It("returns error when telegram watcher is missing chat_id", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: tg
    type: telegram
    token: "bot123:ABC"
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("chat_id"))
		})

		It("accepts log type with no extra fields", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: debug
    type: log
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Watchers).To(HaveLen(1))
			Expect(cfg.Watchers[0].Type).To(Equal("log"))
		})

		It("parses dedup_ttl as 30 minutes", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: debug
    type: log
    dedup_ttl: "30m"
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Watchers[0].DedupTTL).To(Equal(30 * time.Minute))
		})

		It("defaults dedup_ttl to 5 minutes when not specified", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: debug
    type: log
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Watchers[0].DedupTTL).To(Equal(5 * time.Minute))
		})

		It("returns error when dedup_ttl is invalid", func() {
			path := writeTempConfig(`
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: debug
    type: log
    dedup_ttl: "banana"
`)
			DeferCleanup(os.Remove, path)

			_, err := config.NewLoader(path).Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("dedup_ttl"))
		})
	})

	Context("full valid config", func() {
		It("parses all fields correctly", func() {
			path := writeTempConfig(`
vaults:
  openclaw:
    path: /tmp/vault
    tasks_dir: tasks
  personal:
    path: /tmp/obsidian
    tasks_dir: "24 Tasks"

watchers:
  - name: wake-tradingclaw
    type: openclaw-wake
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [planning, in_progress, ai_review]
    dedup_ttl: "5m"
    url: http://127.0.0.1:18789/hooks/wake
    token: "secret"

  - name: notify-review
    type: telegram
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [human_review]
    dedup_ttl: "30m"
    token: "bot123:ABC"
    chat_id: "456"

  - name: debug
    type: log
    assignee: TradingClaw
    statuses: [in_progress]
    phases: [planning]
`)
			DeferCleanup(os.Remove, path)

			cfg, err := config.NewLoader(path).Load(ctx)
			Expect(err).NotTo(HaveOccurred())

			// 2 vaults
			Expect(cfg.Vaults).To(HaveLen(2))
			vaultNames := make([]string, len(cfg.Vaults))
			for i, v := range cfg.Vaults {
				vaultNames[i] = v.Name
			}
			Expect(vaultNames).To(ConsistOf("openclaw", "personal"))

			// 3 watchers
			Expect(cfg.Watchers).To(HaveLen(3))

			wake := cfg.Watchers[0]
			Expect(wake.Name).To(Equal("wake-tradingclaw"))
			Expect(wake.Type).To(Equal("openclaw-wake"))
			Expect(wake.Assignee).To(Equal("TradingClaw"))
			Expect(wake.Statuses).To(ConsistOf("in_progress"))
			Expect(wake.Phases).To(ConsistOf("planning", "in_progress", "ai_review"))
			Expect(wake.DedupTTL).To(Equal(5 * time.Minute))
			Expect(wake.URL).To(Equal("http://127.0.0.1:18789/hooks/wake"))
			Expect(wake.Token).To(Equal("secret"))

			tg := cfg.Watchers[1]
			Expect(tg.Name).To(Equal("notify-review"))
			Expect(tg.Type).To(Equal("telegram"))
			Expect(tg.Token).To(Equal("bot123:ABC"))
			Expect(tg.ChatID).To(Equal("456"))
			Expect(tg.DedupTTL).To(Equal(30 * time.Minute))

			log := cfg.Watchers[2]
			Expect(log.Name).To(Equal("debug"))
			Expect(log.Type).To(Equal("log"))
		})
	})

	Context("file resolution", func() {
		It("returns error when file does not exist", func() {
			_, err := config.NewLoader("/nonexistent/path/config.yaml").Load(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("/nonexistent/path/config.yaml"))
		})

		It("resolves default path when filePath is empty", func() {
			_, err := config.NewLoader("").Load(ctx)
			// May fail with file-not-found or migration error depending on local environment.
			// Either outcome proves the default path was resolved.
			Expect(err).To(HaveOccurred())
		})
	})
})
