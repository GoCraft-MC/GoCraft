package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"GoCraft/config"
	"GoCraft/server"
)

// version is overridden at build time via -ldflags:
//
//	go build -ldflags="-X main.version=v1.2.3" .
var version = "dev"

func main() {
	configPath := flag.String("config", "server.yml", "path to server configuration YAML file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("GoCraft", version)
		return
	}

	// Structured logging to stdout — Pterodactyl captures this stream.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("GoCraft starting", "version", version)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "path", *configPath, "err", err)
		os.Exit(1)
	}

	// Environment variables override YAML values.
	// This is the primary mechanism for Pterodactyl to inject port assignments
	// and feature flags without modifying server.yml on disk.
	if err := cfg.ApplyEnvOverrides(); err != nil {
		slog.Error("invalid environment override", "err", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to initialise server", "err", err)
		os.Exit(1)
	}

	// Graceful shutdown on SIGINT (Ctrl-C) or SIGTERM (Pterodactyl/systemd).
	// signal.NotifyContext cancels ctx when either signal arrives, which
	// propagates through srv.Run → listeners close → world flushes to disk.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}

	slog.Info("server shut down cleanly")
}
