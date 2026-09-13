// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli_test

import (
	"bytes"
	"context"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/task-watcher/pkg/cli"
	"github.com/bborbe/task-watcher/pkg/config"
)

func writeTempConfig(content string) string {
	f, err := os.CreateTemp("", "cli-config-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = f.WriteString(content)
	Expect(err).NotTo(HaveOccurred())
	Expect(f.Close()).To(Succeed())
	return f.Name()
}

const validConfig = `
vaults:
  personal:
    path: /tmp/vault
    tasks_dir: Tasks
watchers:
  - name: review
    assignee: alice
    phases: [human_review]
`

// unsetenv removes name for the duration of the current spec and restores its
// previous state afterwards. GinkgoT().Setenv cannot express "unset".
func unsetenv(name string) {
	original, existed := os.LookupEnv(name)
	Expect(os.Unsetenv(name)).To(Succeed())
	DeferCleanup(func() {
		if existed {
			Expect(os.Setenv(name, original)).To(Succeed())
			return
		}
		Expect(os.Unsetenv(name)).To(Succeed())
	})
}

var _ = Describe("Run", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("returns error when config file does not exist", func() {
		err := cli.Run(ctx, []string{"--config", "/nonexistent/path/config.yaml"})
		Expect(err).To(HaveOccurred())
	})

	It("returns error naming KAFKA_BROKERS when it is unset", func() {
		path := writeTempConfig(validConfig)
		DeferCleanup(os.Remove, path)
		unsetenv(config.EnvKafkaBrokers)
		GinkgoT().Setenv(config.EnvTopicPrefix, "develop")

		err := cli.Run(ctx, []string{"--config", path})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(config.EnvKafkaBrokers))
	})

	It("returns error naming TOPIC_PREFIX when it is empty", func() {
		path := writeTempConfig(validConfig)
		DeferCleanup(os.Remove, path)
		GinkgoT().Setenv(config.EnvKafkaBrokers, "127.0.0.1:9092")
		GinkgoT().Setenv(config.EnvTopicPrefix, "")

		err := cli.Run(ctx, []string{"--config", path})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(config.EnvTopicPrefix))
	})

	It("--help output contains --config and --verbose but not alsologtostderr", func() {
		// Capture stdout because cobra writes help to os.Stdout by default
		origStdout := os.Stdout
		r, w, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		os.Stdout = w

		_ = cli.Run(ctx, []string{"--help"})

		Expect(w.Close()).To(Succeed())
		os.Stdout = origStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		out := buf.String()

		Expect(out).To(ContainSubstring("--config"))
		Expect(out).To(ContainSubstring("--verbose"))
		Expect(out).To(ContainSubstring("~/.config/task-watcher/config.yaml"))
		Expect(out).NotTo(ContainSubstring("$XDG_CONFIG_HOME"))
		Expect(strings.Contains(out, "alsologtostderr")).To(BeFalse())
	})
})
