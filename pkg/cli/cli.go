// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bborbe/errors"
	"github.com/spf13/cobra"

	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/factory"
)

var version = "dev"

// Execute is the entry point called from main. It creates a root context with
// signal handling and delegates to Run.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	if err := Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Run builds and executes the cobra command tree with the given args.
func Run(ctx context.Context, args []string) error {
	var configPath string
	var verbose bool

	rootCmd := &cobra.Command{
		Use:   "task-watcher",
		Short: "Watches vault task files and notifies agents via webhook",
		Long: `Watches vault task files and notifies agents via webhook.

Configuration: reads ~/.config/task-watcher/config.yaml (XDG), falling back to ~/.task-watcher/config.yaml (legacy). Override with --config.`,
		Version:      version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if verbose {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}

			loader := factory.CreateConfigLoader(configPath)
			cfg, err := loader.Load(ctx)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			if err := logStartup(ctx, cfg); err != nil {
				return err
			}

			notifiers := factory.CreateNotifiers(cfg)
			w := factory.CreateWatcher(cfg, notifiers)

			errCh := make(chan error, 1)
			go func() {
				errCh <- w.Watch(ctx)
			}()

			select {
			case <-ctx.Done():
				slog.Info("shutting down")
				select {
				case <-errCh:
					return nil
				case <-time.After(5 * time.Second):
					slog.Error("shutdown timed out")
					return fmt.Errorf("shutdown timed out")
				}
			case err := <-errCh:
				if err != nil && err != context.Canceled {
					return fmt.Errorf("watcher error: %w", err)
				}
				return nil
			}
		},
	}

	rootCmd.Flags().
		StringVar(&configPath, "config", "", "path to config YAML file (default: ~/.config/task-watcher/config.yaml, fallback: ~/.task-watcher/config.yaml)")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "enable debug logging")

	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(ctx)
}

// logStartup loads the publish destination from the environment and logs the
// effective startup configuration. It returns an error when the destination is
// not configured, so a missing variable surfaces as a non-zero exit naming it.
func logStartup(ctx context.Context, cfg config.Config) error {
	destination, err := config.LoadDestinationFromEnv(ctx)
	if err != nil {
		return errors.Wrapf(ctx, err, "load destination")
	}

	slog.Info("task-watcher starting", "version", version)
	slog.Info(
		"publish destination",
		"kafka_brokers",
		destination.KafkaBrokers.String(),
		"topic_prefix",
		destination.TopicPrefix.String(),
	)
	for _, v := range cfg.Vaults {
		slog.Info("watching vault", "name", v.Name, "path", v.Path, "tasksDir", v.TasksDir)
	}
	for _, w := range cfg.Watchers {
		slog.Info(
			"configured watcher",
			"name",
			w.Name,
			"assignee",
			w.Assignee,
			"statuses",
			w.Statuses,
			"phases",
			w.Phases,
			"dedupTTL",
			w.DedupTTL,
		)
	}
	return nil
}
