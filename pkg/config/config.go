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

// WatcherConfig holds the configuration for a single watcher entry.
type WatcherConfig struct {
	Name     string
	Type     string
	Assignee string
	Statuses []string
	Phases   []string
	DedupTTL time.Duration
	// openclaw-wake fields
	URL   string
	Token string
	// telegram fields
	ChatID string
}

// Config holds the parsed task-watcher configuration.
type Config struct {
	Vaults   []VaultConfig
	Watchers []WatcherConfig
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

type rawWatcherEntry struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Assignee string   `yaml:"assignee"`
	Statuses []string `yaml:"statuses"`
	Phases   []string `yaml:"phases"`
	DedupTTL string   `yaml:"dedup_ttl"`
	URL      string   `yaml:"url"`
	Token    string   `yaml:"token"`
	ChatID   string   `yaml:"chat_id"`
}

type rawConfig struct {
	Vaults   map[string]rawVaultEntry `yaml:"vaults"`
	Watchers []rawWatcherEntry        `yaml:"watchers"`
	// Old flat fields — present only for migration error detection
	Assignee     string   `yaml:"assignee"`
	Webhook      string   `yaml:"webhook"`
	Format       string   `yaml:"format"`
	WebhookToken string   `yaml:"webhook_token"`
	Statuses     []string `yaml:"statuses"`
	Phases       []string `yaml:"phases"`
	OldDedupTTL  string   `yaml:"dedup_ttl"`
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

func validateWatcherType(ctx context.Context, rw rawWatcherEntry) error {
	switch rw.Type {
	case "openclaw-wake":
		if rw.URL == "" {
			return errors.Errorf(
				ctx,
				"watcher %q (openclaw-wake): missing required field: url",
				rw.Name,
			)
		}
		if rw.Token == "" {
			return errors.Errorf(
				ctx,
				"watcher %q (openclaw-wake): missing required field: token",
				rw.Name,
			)
		}
	case "telegram":
		if rw.Token == "" {
			return errors.Errorf(
				ctx,
				"watcher %q (telegram): missing required field: token",
				rw.Name,
			)
		}
		if rw.ChatID == "" {
			return errors.Errorf(
				ctx,
				"watcher %q (telegram): missing required field: chat_id",
				rw.Name,
			)
		}
	case "log":
		// no extra fields required
	default:
		return errors.Errorf(
			ctx,
			"watcher %q: unknown type %q (must be openclaw-wake, telegram, or log)",
			rw.Name,
			rw.Type,
		)
	}
	return nil
}

func parseDedupTTL(ctx context.Context, rw rawWatcherEntry) (time.Duration, error) {
	if rw.DedupTTL == "" {
		return 5 * time.Minute, nil
	}
	parsed, err := time.ParseDuration(rw.DedupTTL)
	if err != nil {
		return 0, errors.Wrapf(ctx, err, "watcher %q: parse dedup_ttl %q", rw.Name, rw.DedupTTL)
	}
	return parsed, nil
}

func parseWatchers(ctx context.Context, rawList []rawWatcherEntry) ([]WatcherConfig, error) {
	watchers := make([]WatcherConfig, 0, len(rawList))
	for i, rw := range rawList {
		if rw.Name == "" {
			return nil, errors.Errorf(ctx, "watcher[%d]: missing required field: name", i)
		}
		if rw.Type == "" {
			return nil, errors.Errorf(ctx, "watcher %q: missing required field: type", rw.Name)
		}
		if err := validateWatcherType(ctx, rw); err != nil {
			return nil, err
		}
		dedupTTL, err := parseDedupTTL(ctx, rw)
		if err != nil {
			return nil, err
		}

		watchers = append(watchers, WatcherConfig{
			Name:     rw.Name,
			Type:     rw.Type,
			Assignee: rw.Assignee,
			Statuses: rw.Statuses,
			Phases:   rw.Phases,
			DedupTTL: dedupTTL,
			URL:      rw.URL,
			Token:    rw.Token,
			ChatID:   rw.ChatID,
		})
	}
	return watchers, nil
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

	if raw.Assignee != "" || raw.Webhook != "" || raw.Format != "" || raw.WebhookToken != "" {
		return Config{}, errors.Errorf(ctx,
			"config uses the old flat format (assignee/webhook/format fields at the top level). "+
				"Please migrate to the new watchers list format. See CHANGELOG for migration instructions.")
	}

	if len(raw.Vaults) == 0 {
		return Config{}, errors.Errorf(ctx, "missing required field: vaults")
	}

	vaults, err := parseVaults(ctx, raw.Vaults)
	if err != nil {
		return Config{}, err
	}

	watchers, err := parseWatchers(ctx, raw.Watchers)
	if err != nil {
		return Config{}, err
	}

	return Config{Vaults: vaults, Watchers: watchers}, nil
}
