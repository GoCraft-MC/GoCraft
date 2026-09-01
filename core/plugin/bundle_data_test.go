package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

// The three calls below are the sequence the server performs at boot, spelled
// out rather than hidden behind a helper: a wrapper that only the tests used
// was a second load path free to drift from the one that ships.
func TestScanPreflightLoadPreparesPluginData(t *testing.T) {
	directory := t.TempDir()
	writeBundle(t, directory, "protect.gcpkg", `
id = "dev.example.protect"
version = "1.0.0"
api = 1
runtime = "recording"
`, map[string]string{
		"config/config.yml":        "enabled: true\n",
		"config/lang/messages.yml": "denied: Protected area\n",
		"payload/plugin":           "not extracted",
	})
	var order []string
	var loaded []Bundle
	registry := NewRegistry(context.Background(), 0, nil, nil)
	if err := registry.RegisterRuntime(&recordingRuntime{order: &order, loaded: &loaded}); err != nil {
		t.Fatal(err)
	}
	bundles, err := ScanBundles(directory)
	if err != nil {
		t.Fatalf("ScanBundles(%q): %v", directory, err)
	}
	if err := registry.Preflight(context.Background(), bundles); err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if err := registry.LoadAll(context.Background(), bundles); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	wantDirectory := filepath.Join(directory, "dev.example.protect")
	if len(loaded) != 1 {
		t.Fatalf("loaded bundle count = %d", len(loaded))
	}
	if loaded[0].DataDirectory != wantDirectory {
		t.Fatalf("runtime data directory = %q", loaded[0].DataDirectory)
	}
	assertFileContents(t, filepath.Join(wantDirectory, "config.yml"), "enabled: true\n")
	assertFileContents(t, filepath.Join(wantDirectory, "lang", "messages.yml"), "denied: Protected area\n")
	if _, err := os.Stat(filepath.Join(wantDirectory, "payload", "plugin")); !os.IsNotExist(err) {
		t.Fatalf("payload was extracted into data directory: %v", err)
	}
}

func TestPrepareBundleDataPreservesExistingConfig(t *testing.T) {
	directory := t.TempDir()
	writeBundle(t, directory, "settings.gcpkg", `
id = "dev.example.settings"
version = "1.0.0"
api = 1
runtime = "recording"
`, map[string]string{"config/config.yml": "from bundle\n"})
	opened, err := gcpkg.Open(filepath.Join(directory, "settings.gcpkg"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareBundleData(Bundle{Bundle: opened})
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(prepared.DataDirectory, "config.yml")
	if err := os.WriteFile(configPath, []byte("server owner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareBundleData(Bundle{Bundle: opened}); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, configPath, "server owner\n")
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != want {
		t.Fatalf("%s = %q, want %q", path, contents, want)
	}
}
