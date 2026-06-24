package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDimensionWorldDirectoriesAreVisibleSiblings(t *testing.T) {
	nether, end := dimensionWorldDirectories(filepath.Join("home", "container", "world"))
	if nether != filepath.Join("home", "container", "world_nether") || end != filepath.Join("home", "container", "world_end") {
		t.Fatalf("dimension directories = %q / %q", nether, end)
	}
}

func TestPrepareDimensionWorldDirectoryCopiesLegacyDataWithoutRemovingIt(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "world", "DIM-1")
	target := filepath.Join(root, "world_nether")
	if err := os.MkdirAll(filepath.Join(legacy, "region"), 0o755); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(legacy, "region", "r.0.0.mca")
	if err := os.WriteFile(sourceFile, []byte("legacy chunk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := prepareDimensionWorldDirectory(target, legacy); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(filepath.Join(target, "region", "r.0.0.mca"))
	if err != nil || string(copied) != "legacy chunk" {
		t.Fatalf("copied chunk = %q, err=%v", copied, err)
	}
	if _, err := os.Stat(sourceFile); err != nil {
		t.Fatalf("legacy data was removed: %v", err)
	}
}
