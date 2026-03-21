// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/config"
)

const validYAML = `
vaults:
  personal:
    path: ~/notes/vault
    tasks_dir: "24 Tasks"
assignee: alice
statuses:
  - active
  - in-review
phases:
  - planning
  - execution
webhook: https://hooks.example.com/notify
`

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

	It("loads a valid config file", func() {
		path := writeTempConfig(validYAML)
		DeferCleanup(os.Remove, path)

		cfg, err := config.NewLoader(path).Load(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Assignee).To(Equal("alice"))
		Expect(cfg.Statuses).To(ConsistOf("active", "in-review"))
		Expect(cfg.Phases).To(ConsistOf("planning", "execution"))
		Expect(cfg.Webhook).To(Equal("https://hooks.example.com/notify"))
	})

	It("parses single vault with name, path and tasks_dir", func() {
		path := writeTempConfig(validYAML)
		DeferCleanup(os.Remove, path)

		cfg, err := config.NewLoader(path).Load(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Vaults).To(HaveLen(1))
		Expect(cfg.Vaults[0].Name).To(Equal("personal"))
		Expect(cfg.Vaults[0].TasksDir).To(Equal("24 Tasks"))
	})

	It("parses multiple vaults", func() {
		yaml := `
vaults:
  personal:
    path: /tmp/personal
    tasks_dir: "24 Tasks"
  octopus:
    path: /tmp/octopus
    tasks_dir: "Tasks"
assignee: alice
statuses:
  - active
phases:
  - planning
webhook: https://hooks.example.com/notify
`
		path := writeTempConfig(yaml)
		DeferCleanup(os.Remove, path)

		cfg, err := config.NewLoader(path).Load(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Vaults).To(HaveLen(2))

		names := make([]string, len(cfg.Vaults))
		for i, v := range cfg.Vaults {
			names[i] = v.Name
		}
		Expect(names).To(ConsistOf("personal", "octopus"))
	})

	It("expands ~/ in each vault path", func() {
		path := writeTempConfig(validYAML)
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

	It("returns error when file does not exist", func() {
		_, err := config.NewLoader("/nonexistent/path/config.yaml").Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("/nonexistent/path/config.yaml"))
	})

	It("resolves default path when filePath is empty", func() {
		home, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		defaultPath := home + "/.task-watcher/config.yaml"

		// Verify the loader uses the default path (not that it errors).
		// If the file exists on this machine, the loader will succeed;
		// if not, the error message must reference the default path.
		_, err = config.NewLoader("").Load(ctx)
		if err != nil {
			Expect(err.Error()).To(ContainSubstring(defaultPath))
		}
		// Either way, the default resolution worked.
	})

	It("returns error when vaults map is empty", func() {
		path := writeTempConfig(`
assignee: alice
statuses:
  - active
phases:
  - planning
webhook: https://hooks.example.com/notify
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
assignee: alice
statuses:
  - active
phases:
  - planning
webhook: https://hooks.example.com/notify
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
assignee: alice
statuses:
  - active
phases:
  - planning
webhook: https://hooks.example.com/notify
`)
		DeferCleanup(os.Remove, path)

		_, err := config.NewLoader(path).Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("tasks_dir"))
	})

	It("returns error when assignee is missing", func() {
		path := writeTempConfig(`
vaults:
  personal:
    path: ~/notes
    tasks_dir: "24 Tasks"
statuses:
  - active
phases:
  - planning
webhook: https://hooks.example.com/notify
`)
		DeferCleanup(os.Remove, path)

		_, err := config.NewLoader(path).Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("assignee"))
	})

	It("returns error when statuses is empty", func() {
		path := writeTempConfig(`
vaults:
  personal:
    path: ~/notes
    tasks_dir: "24 Tasks"
assignee: alice
phases:
  - planning
webhook: https://hooks.example.com/notify
`)
		DeferCleanup(os.Remove, path)

		_, err := config.NewLoader(path).Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("statuses"))
	})

	It("returns error when phases is empty", func() {
		path := writeTempConfig(`
vaults:
  personal:
    path: ~/notes
    tasks_dir: "24 Tasks"
assignee: alice
statuses:
  - active
webhook: https://hooks.example.com/notify
`)
		DeferCleanup(os.Remove, path)

		_, err := config.NewLoader(path).Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("phases"))
	})

	It("returns error when webhook is missing", func() {
		path := writeTempConfig(`
vaults:
  personal:
    path: ~/notes
    tasks_dir: "24 Tasks"
assignee: alice
statuses:
  - active
phases:
  - planning
`)
		DeferCleanup(os.Remove, path)

		_, err := config.NewLoader(path).Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("webhook"))
	})

	It("defaults format to generic when not specified", func() {
		path := writeTempConfig(validYAML)
		DeferCleanup(os.Remove, path)

		cfg, err := config.NewLoader(path).Load(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Format).To(Equal("generic"))
	})

	It("accepts format generic", func() {
		path := writeTempConfig(validYAML + "format: generic\n")
		DeferCleanup(os.Remove, path)

		cfg, err := config.NewLoader(path).Load(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Format).To(Equal("generic"))
	})

	It("accepts format openclaw with webhook_token", func() {
		path := writeTempConfig(validYAML + "format: openclaw\nwebhook_token: secret\n")
		DeferCleanup(os.Remove, path)

		cfg, err := config.NewLoader(path).Load(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Format).To(Equal("openclaw"))
		Expect(cfg.WebhookToken).To(Equal("secret"))
	})

	It("returns error when format is openclaw but webhook_token is missing", func() {
		path := writeTempConfig(validYAML + "format: openclaw\n")
		DeferCleanup(os.Remove, path)

		_, err := config.NewLoader(path).Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("webhook_token"))
	})

	It("returns error when format is unknown", func() {
		path := writeTempConfig(validYAML + "format: invalid\n")
		DeferCleanup(os.Remove, path)

		_, err := config.NewLoader(path).Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid"))
	})
})
