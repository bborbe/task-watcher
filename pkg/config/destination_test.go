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

var _ = Describe("LoadDestinationFromEnv", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("parses brokers and preserves the topic prefix when both variables are set", func() {
		GinkgoT().Setenv(config.EnvKafkaBrokers, "127.0.0.1:9092,127.0.0.1:9093")
		GinkgoT().Setenv(config.EnvTopicPrefix, "develop")

		destination, err := config.LoadDestinationFromEnv(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(destination.KafkaBrokers).To(HaveLen(2))
		Expect(destination.TopicPrefix.String()).To(Equal("develop"))
	})

	It("returns error naming KAFKA_BROKERS when it is empty", func() {
		GinkgoT().Setenv(config.EnvKafkaBrokers, "")
		GinkgoT().Setenv(config.EnvTopicPrefix, "develop")

		_, err := config.LoadDestinationFromEnv(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(config.EnvKafkaBrokers))
	})

	It("returns error naming KAFKA_BROKERS when it is unset", func() {
		unsetenv(config.EnvKafkaBrokers)
		GinkgoT().Setenv(config.EnvTopicPrefix, "develop")

		_, err := config.LoadDestinationFromEnv(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(config.EnvKafkaBrokers))
	})

	It("returns error naming TOPIC_PREFIX when it is empty", func() {
		GinkgoT().Setenv(config.EnvKafkaBrokers, "127.0.0.1:9092")
		GinkgoT().Setenv(config.EnvTopicPrefix, "")

		_, err := config.LoadDestinationFromEnv(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(config.EnvTopicPrefix))
	})

	It("returns error naming TOPIC_PREFIX when it is unset", func() {
		GinkgoT().Setenv(config.EnvKafkaBrokers, "127.0.0.1:9092")
		unsetenv(config.EnvTopicPrefix)

		_, err := config.LoadDestinationFromEnv(ctx)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(config.EnvTopicPrefix))
	})
})
