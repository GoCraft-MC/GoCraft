package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"GoCraft/config"
	"GoCraft/server"
)

const configPath = "server.yml"

func main() {
	// Structured logging to stdout.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	cfg, err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load configuration", "path", configPath, "err", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to initialise server", "err", err)
		os.Exit(1)
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}

	slog.Info("server shut down cleanly")
}
