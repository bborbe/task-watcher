// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/bborbe/errors"
	"gopkg.in/yaml.v3"
)

// Config holds the parsed task-watcher configuration.
type Config struct {
	VaultPath    string
	Assignee     string
	Statuses     []string
	Webhook      string
	Phases       []string
	Format       string
	WebhookToken string
}

//counterfeiter:generate -o ../../mocks/config_loader.go --fake-name FakeConfigLoader . Loader

// Loader loads configuration from a file.
type Loader interface {
	Load(ctx context.Context) (Config, error)
}

// NewLoader returns a Loader that reads config from filePath.
func NewLoader(filePath string) Loader {
	return &loader{filePath: filePath}
}

type loader struct {
	filePath string
}

type rawConfig struct {
	Vault struct {
		Path string `yaml:"path"`
	} `yaml:"vault"`
	Assignee     string   `yaml:"assignee"`
	Statuses     []string `yaml:"statuses"`
	Phases       []string `yaml:"phases"`
	Webhook      string   `yaml:"webhook"`
	Format       string   `yaml:"format"`
	WebhookToken string   `yaml:"webhook_token"`
}

func (l *loader) Load(ctx context.Context) (Config, error) {
	filePath := l.filePath
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return Config{}, errors.Wrapf(ctx, err, "get user home dir")
		}
		filePath = filepath.Join(homeDir, ".task-watcher", "config.yaml")
	}

	data, err := os.ReadFile(
		filePath,
	) // #nosec G304 -- filePath is either user-provided CLI arg or the well-known default ~/.task-watcher/config.yaml
	if err != nil {
		return Config{}, errors.Wrapf(ctx, err, "read config file %s", filePath)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, errors.Wrapf(ctx, err, "parse config file %s", filePath)
	}

	if raw.Vault.Path == "" {
		return Config{}, errors.Errorf(ctx, "missing required field: vault.path")
	}
	if raw.Assignee == "" {
		return Config{}, errors.Errorf(ctx, "missing required field: assignee")
	}
	if len(raw.Statuses) == 0 {
		return Config{}, errors.Errorf(ctx, "missing required field: statuses")
	}
	if len(raw.Phases) == 0 {
		return Config{}, errors.Errorf(ctx, "missing required field: phases")
	}
	if raw.Webhook == "" {
		return Config{}, errors.Errorf(ctx, "missing required field: webhook")
	}

	format := raw.Format
	if format == "" {
		format = "generic"
	}
	if format != "generic" && format != "openclaw" {
		return Config{}, errors.Errorf(
			ctx,
			"invalid format %q: must be \"generic\" or \"openclaw\"",
			format,
		)
	}
	if format == "openclaw" && raw.WebhookToken == "" {
		return Config{}, errors.Errorf(ctx, "webhook_token is required when format is \"openclaw\"")
	}

	vaultPath := raw.Vault.Path
	if strings.HasPrefix(vaultPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, errors.Wrapf(ctx, err, "get user home dir")
		}
		vaultPath = home + vaultPath[1:]
	}

	return Config{
		VaultPath:    vaultPath,
		Assignee:     raw.Assignee,
		Statuses:     raw.Statuses,
		Phases:       raw.Phases,
		Webhook:      raw.Webhook,
		Format:       format,
		WebhookToken: raw.WebhookToken,
	}, nil
}
