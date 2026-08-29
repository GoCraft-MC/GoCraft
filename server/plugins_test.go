package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"GoCraft/config"
	coreplugin "GoCraft/core/plugin"
)

func pluginServer(directory string, enabled bool) *Server {
	return &Server{
		cfg: &config.Config{Plugins: config.PluginsConfig{
			Enabled:           enabled,
			Directory:         directory,
			EventBudgetMillis: 2,
		}},
		pluginRegistry: coreplugin.NewRegistry(context.Background(), 2*time.Millisecond, nil, nil),
	}
}

func writeBrokenBundle(t *testing.T, directory string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, "broken.gcpkg"), []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPluginsAcceptsAMissingDirectory(t *testing.T) {
	s := pluginServer(filepath.Join(t.TempDir(), "plugins"), true)
	if err := s.loadPlugins(context.Background()); err != nil {
		t.Fatalf("loadPlugins() = %v, want nil for a missing directory", err)
	}
}

func TestLoadPluginsIsSkippedWhenDisabled(t *testing.T) {
	directory := t.TempDir()
	writeBrokenBundle(t, directory)
	s := pluginServer(directory, false)
	if err := s.loadPlugins(context.Background()); err != nil {
		t.Fatalf("loadPlugins() = %v, want nil when plugins are disabled", err)
	}
}

func TestLoadPluginsRefusesToBootOnAnUnreadableBundle(t *testing.T) {
	directory := t.TempDir()
	writeBrokenBundle(t, directory)
	s := pluginServer(directory, true)
	if err := s.loadPlugins(context.Background()); err == nil {
		t.Fatal("loadPlugins() accepted an unreadable bundle")
	}
}

func TestUnloadPluginsToleratesAnEmptyRegistry(t *testing.T) {
	s := pluginServer(filepath.Join(t.TempDir(), "plugins"), true)
	if err := s.loadPlugins(context.Background()); err != nil {
		t.Fatal(err)
	}
	s.unloadPlugins()
	s.unloadPlugins()
}

func TestUnloadPluginsToleratesAServerWithoutARegistry(t *testing.T) {
	s := &Server{}
	s.unloadPlugins()
}