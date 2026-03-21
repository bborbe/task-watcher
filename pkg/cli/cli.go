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

	"github.com/spf13/cobra"

	"github.com/bborbe/task-watcher/pkg/config"
	"github.com/bborbe/task-watcher/pkg/factory"
	"github.com/bborbe/task-watcher/pkg/notify"
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
	var dryRun bool

	rootCmd := &cobra.Command{
		Use:          "task-watcher",
		Short:        "Watches vault task files and notifies agents via webhook",
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

			slog.Info(
				"task-watcher starting",
				"vaultPath",
				cfg.VaultPath,
				"assignee",
				cfg.Assignee,
				"dryRun",
				dryRun,
			)

			notifier := buildNotifier(cfg, dryRun)
			w := factory.CreateWatcher(cfg, notifier)

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
		StringVar(&configPath, "config", "", "path to config YAML file (default: ~/.task-watcher/config.yaml)")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "enable debug logging")
	rootCmd.Flags().
		BoolVar(&dryRun, "dry-run", false, "log notifications instead of sending HTTP webhooks")

	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(ctx)
}

// buildNotifier selects the appropriate notifier based on format and dry-run mode.
func buildNotifier(cfg config.Config, dryRun bool) notify.Notifier {
	if cfg.Format == "openclaw" {
		if dryRun {
			return factory.CreateDryRunOpenClawNotifier(cfg)
		}
		return factory.CreateOpenClawNotifier(cfg)
	}
	if dryRun {
		return factory.CreateDryRunNotifier(cfg)
	}
	return factory.CreateNotifier(cfg)
}
