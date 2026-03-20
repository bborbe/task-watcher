// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"context"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/config"
)

const validYAML = `
vault:
  path: ~/notes/vault
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

	It("expands ~/ in vault path", func() {
		path := writeTempConfig(validYAML)
		DeferCleanup(os.Remove, path)

		cfg, err := config.NewLoader(path).Load(ctx)
		Expect(err).NotTo(HaveOccurred())

		home, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.VaultPath).To(HavePrefix(home))
		Expect(cfg.VaultPath).NotTo(ContainSubstring("~"))
		Expect(strings.HasPrefix(cfg.VaultPath, home+"/notes/vault")).To(BeTrue())
	})

	It("returns error when file does not exist", func() {
		_, err := config.NewLoader("/nonexistent/path/config.yaml").Load(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("/nonexistent/path/config.yaml"))
	})

	It("returns error when vault.path is missing", func() {
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
		Expect(err.Error()).To(ContainSubstring("vault.path"))
	})

	It("returns error when assignee is missing", func() {
		path := writeTempConfig(`
vault:
  path: ~/notes
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
vault:
  path: ~/notes
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
vault:
  path: ~/notes
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
vault:
  path: ~/notes
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
})
