package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"context"
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

func initConfig(cmd *cobra.Command) error {
	viper.SetEnvPrefix("GOCRAFT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	return viper.BindPFlags(cmd.Flags())
}

var RootCommand = &cobra.Command{
	Use:   "gocraft",
	Short: "A minecraft server written in go",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cmd)
	},
	RunE: run,
}

func init() {
	RootCommand.PersistentFlags().StringP("config", "c", "server.yml", "path to server configuration YAML file")
	RootCommand.PersistentFlags().StringP("log-dir", "", "logs", "directory for latest.log and compressed log archives")
	RootCommand.PersistentFlags().Int("max-log-files", 10, "maximum number of compressed log archives to keep")
}

func initLogger() {
	logDirectory := viper.GetString("log-dir")
	maxLogArchives := viper.GetInt("max-log-files")

	// Keep stdout logging for Pterodactyl, then tee the same output to the
	// Paper-style logs/latest.log file when file logging is available.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	logFile, err := serverlog.Open(logDirectory, maxLogArchives)
	if err != nil {
		slog.Error("file logging disabled", "directory", logDirectory, "err", err)
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
}

func loadConfig() (*config.Config, error) {
	configPath := viper.GetString("config")
	return config.Load(configPath)
}

func run(cmd *cobra.Command, args []string) error {
	initLogger()
	cfg, err := loadConfig()
	slog.Info("GoCraft starting", "version", server.GetVersion())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	// Environment variables override YAML values.
	// This is the primary mechanism for Pterodactyl to inject port assignments
	// and feature flags without modifying server.yml on disk.
	if err := cfg.ApplyEnvOverrides(); err != nil {
		return fmt.Errorf("invalid environment override: %v", err)
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
		return fmt.Errorf("failed to initialize server: %v", err)
	}

	// Graceful shutdown on SIGINT (Ctrl-C) or SIGTERM (Pterodactyl/systemd).
	// signal.NotifyContext cancels ctx when either signal arrives, which
	// propagates through srv.Run → listeners close → world flushes to disk.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("server stopped with error: %v", err)
	}

	slog.Info("server shut down cleanly")
	return nil
}
