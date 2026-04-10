package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"GoCraft/config"
	"GoCraft/internal/debuglog"
	"GoCraft/internal/protocoldata"
	"GoCraft/internal/serverlog"
	javaworld "GoCraft/java/world"
	"GoCraft/server"
)

// version is overridden at build time via -ldflags:
//
//	go build -ldflags="-X main.version=v1.2.3" .
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "server.yml", "path to server configuration YAML file")
	logDirectory := flag.String("log-dir", "logs", "directory for latest.log and compressed log archives")
	maxLogArchives := flag.Int("max-log-files", 10, "maximum number of compressed log archives to keep")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("GoCraft", version)
		return 0
	}

	// Keep stdout logging for Pterodactyl, then tee the same output to the
	// Paper-style logs/latest.log file when file logging is available.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	logFile, err := serverlog.Open(*logDirectory, *maxLogArchives)
	if err != nil {
		slog.Error("file logging disabled", "directory", *logDirectory, "err", err)
	} else {
		defer func() {
			if err := logFile.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "failed to close latest.log:", err)
			}
		}()
		slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
	}

	slog.Info("GoCraft starting", "version", version)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load configuration", "path", *configPath, "err", err)
		return 1
	}

	// Environment variables override YAML values.
	// This is the primary mechanism for Pterodactyl to inject port assignments
	// and feature flags without modifying server.yml on disk.
	if err := cfg.ApplyEnvOverrides(); err != nil {
		slog.Error("invalid environment override", "err", err)
		return 1
	}
	debuglog.Configure(debuglog.Settings{
		StartupRegistry:    cfg.Debug.StartupRegistry,
		WorldLoading:       cfg.Debug.WorldLoading,
		MobSpawning:        cfg.Debug.MobSpawning,
		Autosaves:          cfg.Debug.Autosaves,
		EntityEvents:       cfg.Debug.EntityEvents,
		EntityTickOverruns: cfg.Debug.EntityTickOverruns,
		BedrockCatalogues:  cfg.Debug.BedrockCatalogues,
		BedrockLogin:       cfg.Debug.BedrockLogin,
		BedrockChunks:      cfg.Debug.BedrockChunks,
		BedrockInventory:   cfg.Debug.BedrockInventory,
		Profiling:          cfg.Debug.Profiling,
	})
	if debuglog.Enabled(debuglog.StartupRegistry) {
		protocolVersion, packetCount := protocoldata.StartupSummary()
		slog.Info("protocoldata: loaded protocol packet IDs", "version", protocolVersion, "packets", packetCount)
		javaworld.LogStartupSummary()
	}

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("failed to initialise server", "err", err)
		return 1
	}

	// Graceful shutdown on SIGINT (Ctrl-C) or SIGTERM (Pterodactyl/systemd).
	// signal.NotifyContext cancels ctx when either signal arrives, which
	// propagates through srv.Run → listeners close → world flushes to disk.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		slog.Error("server stopped with error", "err", err)
		return 1
	}

	slog.Info("server shut down cleanly")
	return 0
}
