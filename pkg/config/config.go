// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bborbe/errors"
	"gopkg.in/yaml.v3"
)

// VaultConfig holds the configuration for a single vault.
type VaultConfig struct {
	Name     string
	Path     string
	TasksDir string
}

// Config holds the parsed task-watcher configuration.
type Config struct {
	Vaults       []VaultConfig
	Assignee     string
	Statuses     []string
	Webhook      string
	Phases       []string
	Format       string
	WebhookToken string
	DedupTTL     time.Duration
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

type rawVaultEntry struct {
	Path     string `yaml:"path"`
	TasksDir string `yaml:"tasks_dir"`
}

type rawConfig struct {
	Vaults       map[string]rawVaultEntry `yaml:"vaults"`
	Assignee     string                   `yaml:"assignee"`
	Statuses     []string                 `yaml:"statuses"`
	Phases       []string                 `yaml:"phases"`
	Webhook      string                   `yaml:"webhook"`
	Format       string                   `yaml:"format"`
	WebhookToken string                   `yaml:"webhook_token"`
	DedupTTL     string                   `yaml:"dedup_ttl"`
}

func resolveFilePath(filePath string) (string, error) {
	if filePath != "" {
		return filePath, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".task-watcher", "config.yaml"), nil
}

func parseFormat(ctx context.Context, raw rawConfig) (string, error) {
	format := raw.Format
	if format == "" {
		format = "generic"
	}
	if format != "generic" && format != "openclaw" {
		return "", errors.Errorf(
			ctx,
			"invalid format %q: must be \"generic\" or \"openclaw\"",
			format,
		)
	}
	if format == "openclaw" && raw.WebhookToken == "" {
		return "", errors.Errorf(ctx, "webhook_token is required when format is \"openclaw\"")
	}
	return format, nil
}

func parseVaults(ctx context.Context, rawVaults map[string]rawVaultEntry) ([]VaultConfig, error) {
	var homeDir string
	vaults := make([]VaultConfig, 0, len(rawVaults))
	for name, entry := range rawVaults {
		if entry.Path == "" {
			return nil, errors.Errorf(ctx, "vault %q missing required field: path", name)
		}
		if entry.TasksDir == "" {
			return nil, errors.Errorf(ctx, "vault %q missing required field: tasks_dir", name)
		}
		vaultPath, err := expandHome(entry.Path, &homeDir)
		if err != nil {
			return nil, errors.Wrapf(ctx, err, "get user home dir")
		}
		vaults = append(vaults, VaultConfig{Name: name, Path: vaultPath, TasksDir: entry.TasksDir})
	}
	sort.Slice(vaults, func(i, j int) bool {
		return vaults[i].Name < vaults[j].Name
	})
	return vaults, nil
}

func expandHome(path string, homeDir *string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	if *homeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		*homeDir = home
	}
	return *homeDir + path[1:], nil
}

func (l *loader) Load(ctx context.Context) (Config, error) {
	filePath, err := resolveFilePath(l.filePath)
	if err != nil {
		return Config{}, errors.Wrapf(ctx, err, "get user home dir")
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

	if len(raw.Vaults) == 0 {
		return Config{}, errors.Errorf(ctx, "missing required field: vaults")
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

	format, err := parseFormat(ctx, raw)
	if err != nil {
		return Config{}, err
	}

	vaults, err := parseVaults(ctx, raw.Vaults)
	if err != nil {
		return Config{}, err
	}

	dedupTTL := 5 * time.Minute // default
	if raw.DedupTTL != "" {
		parsed, err := time.ParseDuration(raw.DedupTTL)
		if err != nil {
			return Config{}, errors.Wrapf(ctx, err, "parse dedup_ttl %q", raw.DedupTTL)
		}
		dedupTTL = parsed
	}

	return Config{
		Vaults:       vaults,
		Assignee:     raw.Assignee,
		Statuses:     raw.Statuses,
		Phases:       raw.Phases,
		Webhook:      raw.Webhook,
		Format:       format,
		WebhookToken: raw.WebhookToken,
		DedupTTL:     dedupTTL,
	}, nil
}
