package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginDefaultsAreEnabledAndPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Plugins.Enabled {
		t.Fatalf("plugins should be enabled by default, got %+v", cfg.Plugins)
	}
	if cfg.Plugins.Directory != "plugins" {
		t.Fatalf("plugins.directory = %q, want plugins", cfg.Plugins.Directory)
	}
	if cfg.Plugins.EventBudgetMillis != 2 {
		t.Fatalf("plugins.event_budget_ms = %d, want 2", cfg.Plugins.EventBudgetMillis)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"plugins:", "directory:", "event_budget_ms:"} {
		if !strings.Contains(string(data), expected) {
			t.Errorf("generated config is missing %q", expected)
		}
	}
}

func TestPluginConfigurationIsValidated(t *testing.T) {
	cfg := defaults()
	cfg.Plugins.Directory = "   "
	if err := cfg.validate(); err == nil {
		t.Fatal("empty plugins.directory was accepted")
	}
	cfg = defaults()
	cfg.Plugins.EventBudgetMillis = 0
	if err := cfg.validate(); err == nil {
		t.Fatal("zero plugins.event_budget_ms was accepted")
	}
	cfg = defaults()
	cfg.Plugins.EventBudgetMillis = 5000
	if err := cfg.validate(); err == nil {
		t.Fatal("out of range plugins.event_budget_ms was accepted")
	}
}

func TestDisabledPluginsSkipValidation(t *testing.T) {
	cfg := defaults()
	cfg.Plugins.Enabled = false
	cfg.Plugins.Directory = ""
	cfg.Plugins.EventBudgetMillis = 0
	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() with plugins disabled = %v", err)
	}
}
