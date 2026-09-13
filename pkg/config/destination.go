// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"os"

	"github.com/bborbe/cqrs/base"
	"github.com/bborbe/errors"
	"github.com/bborbe/kafka"
)

// EnvKafkaBrokers is the environment variable holding the comma-separated Kafka broker list.
const EnvKafkaBrokers = "KAFKA_BROKERS"

// EnvTopicPrefix is the environment variable holding the Kafka topic prefix.
const EnvTopicPrefix = "TOPIC_PREFIX"

// Destination holds the publish destination read from the environment.
type Destination struct {
	KafkaBrokers kafka.Brokers
	TopicPrefix  base.TopicPrefix
}

// LoadDestinationFromEnv reads the publish destination from the environment.
// Both KAFKA_BROKERS (comma-separated) and TOPIC_PREFIX are required; an
// unset or empty variable is an error naming that variable.
func LoadDestinationFromEnv(ctx context.Context) (Destination, error) {
	brokersValue := os.Getenv(EnvKafkaBrokers)
	if brokersValue == "" {
		return Destination{}, errors.Errorf(
			ctx,
			"environment variable %s is not set",
			EnvKafkaBrokers,
		)
	}
	prefixValue := os.Getenv(EnvTopicPrefix)
	if prefixValue == "" {
		return Destination{}, errors.Errorf(
			ctx,
			"environment variable %s is not set",
			EnvTopicPrefix,
		)
	}
	return Destination{
		KafkaBrokers: kafka.ParseBrokersFromString(brokersValue),
		TopicPrefix:  base.TopicPrefix(prefixValue),
	}, nil
}
