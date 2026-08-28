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
	if !cfg.PermissionEditor.Enabled {
		t.Fatalf("permission editor should be enabled by default, got %+v", cfg.PermissionEditor)
	}
	if !strings.HasPrefix(cfg.PermissionEditor.EditorURL, "https://") {
		t.Fatalf("editor_url should be an https URL, got %q", cfg.PermissionEditor.EditorURL)
	}
	if !strings.HasPrefix(cfg.PermissionEditor.BytebinURL, "https://") {
		t.Fatalf("bytebin_url should be an https URL, got %q", cfg.PermissionEditor.BytebinURL)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"permission_editor:", "editor_url:", "bytebin_url:"} {
		if !strings.Contains(string(data), expected) {
			t.Errorf("generated config is missing %q", expected)
		}
	}
}

func TestPermissionEditorConfigurationIsValidated(t *testing.T) {
	cfg := defaults()
	cfg.PermissionEditor.EditorURL = "javascript:alert(1)"
	if err := cfg.validate(); err == nil {
		t.Fatal("unsafe permission editor editor_url was accepted")
	}
	cfg = defaults()
	cfg.PermissionEditor.BytebinURL = "not-a-url"
	if err := cfg.validate(); err == nil {
		t.Fatal("invalid bytebin_url was accepted")
	}
}

func TestPermissionEditorEnvironmentOverrides(t *testing.T) {
	t.Setenv("GOCRAFT_PERMISSION_EDITOR_URL", "https://myeditor.example/editor")
	t.Setenv("GOCRAFT_PERMISSION_EDITOR_BYTEBIN", "https://mybytebin.example")
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if cfg.PermissionEditor.EditorURL != "https://myeditor.example/editor" {
		t.Fatalf("editor URL override = %q", cfg.PermissionEditor.EditorURL)
	}
	if cfg.PermissionEditor.BytebinURL != "https://mybytebin.example" {
		t.Fatalf("bytebin URL override = %q", cfg.PermissionEditor.BytebinURL)
	}
}
