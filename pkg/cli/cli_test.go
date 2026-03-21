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
)

var _ = Describe("Run", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("returns error when --config flag is missing", func() {
		err := cli.Run(ctx, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("config"))
	})

	It("returns error when config file does not exist", func() {
		err := cli.Run(ctx, []string{"--config", "/nonexistent.yaml"})
		Expect(err).To(HaveOccurred())
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
		Expect(strings.Contains(out, "alsologtostderr")).To(BeFalse())
	})
})
