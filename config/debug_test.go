package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedConfigDisablesEveryDebugCategory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Debug != (DebugConfig{}) {
		t.Fatalf("generated debug config = %+v, want every category disabled", cfg.Debug)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, key := range []string{
		"startup_registry",
		"environment_overrides",
		"world_loading",
		"mob_spawning",
		"autosaves",
		"entity_events",
		"entity_tick_overruns",
		"bedrock_catalogues",
		"bedrock_login",
		"bedrock_chunks",
		"bedrock_inventory",
		"profiling",
	} {
		if !strings.Contains(text, key+": false") {
			t.Fatalf("generated server.yml does not contain %s: false\n%s", key, text)
		}
	}
}

func TestLoadEnablesOnlySelectedDebugCategories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yml")
	data := []byte(`debug:
    entity_tick_overruns: true
    bedrock_chunks: true
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Debug.EntityTickOverruns || !cfg.Debug.BedrockChunks {
		t.Fatalf("selected debug categories were not enabled: %+v", cfg.Debug)
	}
	cfg.Debug.EntityTickOverruns = false
	cfg.Debug.BedrockChunks = false
	if cfg.Debug != (DebugConfig{}) {
		t.Fatalf("unselected debug categories unexpectedly enabled: %+v", cfg.Debug)
	}
}

func TestEnvironmentOverrideLogsFollowDebugSwitch(t *testing.T) {
	previousLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	t.Setenv("GOCRAFT_MOTD", "debug-switch-test")

	var output bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatalf("ApplyEnvOverrides() error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("environment override was logged while disabled: %q", output.String())
	}

	output.Reset()
	cfg.Debug.EnvironmentOverrides = true
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatalf("ApplyEnvOverrides() with debug enabled error = %v", err)
	}
	if got := output.String(); !strings.Contains(got, "GOCRAFT_MOTD") || !strings.Contains(got, "debug-switch-test") {
		t.Fatalf("enabled environment override output = %q", got)
	}
}
