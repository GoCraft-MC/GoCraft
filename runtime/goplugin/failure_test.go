package goplugin

import (
	"os"
	"strings"
	"testing"
	"time"

	"GoCraft/core/plugin"
	"GoCraft/runtime/link"
)

func TestRuntimeCleansUpPluginFailures(t *testing.T) {
	for _, phase := range []string{"load", "enable"} {
		t.Run(phase, func(t *testing.T) {
			extractDirectory := t.TempDir()
			socketDirectory, err := os.MkdirTemp("", "gc-go-fail-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(socketDirectory)
			runtime := New(Config{
				ExtractDirectory: extractDirectory, SocketDirectory: socketDirectory,
				StartTimeout: 3 * time.Second,
				Spawn:        func(string) link.Spawn { return helperSpawnFailure(phase) },
			})
			if err := runtime.Start(t.Context(), nil); err != nil {
				t.Fatal(err)
			}
			bundle := plugin.Bundle{
				Path:          writeTestBundleWith(t, "bin/example", []byte("placeholder"), helperCommandTree(t)),
				DataDirectory: t.TempDir(),
				Manifest: plugin.Manifest{
					ID: "example", Entry: "bin/example", CommandTree: commandTreeEntry,
				},
			}
			if _, err := runtime.Load(t.Context(), bundle); err == nil ||
				!strings.Contains(err.Error(), phase+" failure") {
				t.Fatalf("Load() error = %v", err)
			}
			entries, err := os.ReadDir(extractDirectory)
			if err != nil || len(entries) != 0 {
				t.Fatalf("extraction leftovers = %v, %v", entries, err)
			}
			if err := runtime.Stop(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
