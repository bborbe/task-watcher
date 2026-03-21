// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bborbe/task-watcher/pkg/factory"
)

func main() {
	configPath := flag.String("config", "", "path to config YAML file")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "config flag required")
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loader := factory.CreateConfigLoader(*configPath)
	cfg, err := loader.Load(ctx)
	if err != nil {
		slog.Error("failed to load config", "path", *configPath, "error", err)
		os.Exit(1)
	}

	slog.Info("task-watcher starting", "vaultPath", cfg.VaultPath, "assignee", cfg.Assignee)

	notifier := factory.CreateNotifier(cfg)
	w := factory.CreateWatcher(cfg, notifier)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Watch(ctx)
	}()

	select {
	case sig := <-sigCh:
		slog.Info("shutting down", "signal", sig)
		cancel()
		select {
		case <-errCh:
			os.Exit(0)
		case <-time.After(5 * time.Second):
			slog.Error("shutdown timed out")
			os.Exit(1)
		}
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			slog.Error("watcher error", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
