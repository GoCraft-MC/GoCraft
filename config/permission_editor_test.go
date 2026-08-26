package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPermissionEditorDefaultsAreSafeAndPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.PermissionEditor.Enabled || cfg.PermissionEditor.Address != "127.0.0.1:8080" {
		t.Fatalf("permission editor defaults = %+v", cfg.PermissionEditor)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"permission_editor:", "session_minutes: 15"} {
		if !strings.Contains(string(data), expected) {
			t.Errorf("generated config is missing %q", expected)
		}
	}
}

func TestPermissionEditorConfigurationIsValidated(t *testing.T) {
	cfg := defaults()
	cfg.PermissionEditor.PublicURL = "javascript:alert(1)"
	if err := cfg.validate(); err == nil {
		t.Fatal("unsafe permission editor URL was accepted")
	}
	cfg = defaults()
	cfg.PermissionEditor.Address = "missing-port"
	if err := cfg.validate(); err == nil {
		t.Fatal("invalid permission editor bind address was accepted")
	}
}

func TestPermissionEditorEnvironmentOverrides(t *testing.T) {
	t.Setenv("GOCRAFT_PERMISSION_EDITOR_ADDR", "0.0.0.0:9090")
	t.Setenv("GOCRAFT_PERMISSION_EDITOR_URL", "https://permissions.example/")
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if cfg.PermissionEditor.Address != "0.0.0.0:9090" || cfg.PermissionEditor.PublicURL != "https://permissions.example" {
		t.Fatalf("permission editor overrides = %+v", cfg.PermissionEditor)
	}
}
