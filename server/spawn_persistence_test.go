package server

import (
	"os"
	"path/filepath"
	"testing"

	"GoCraft/core/spatial"
)

func TestWorldSpawnPersistenceRoundTrip(t *testing.T) {
	directory := t.TempDir()
	want := spatial.Vec3{X: -12.5, Y: 83, Z: 400.5}
	if err := saveWorldSpawn(directory, want); err != nil {
		t.Fatal(err)
	}
	got, ok := loadSavedWorldSpawn(directory)
	if !ok || got != want {
		t.Fatalf("loaded spawn = %+v, %v; want %+v, true", got, ok, want)
	}
	if _, err := os.Stat(filepath.Join(directory, worldSpawnFile+".tmp")); !os.IsNotExist(err) {
		t.Fatalf("temporary spawn file remains: %v", err)
	}
}

func TestInvalidWorldSpawnIsRejected(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, worldSpawnFile), []byte(`{"X":"bad"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadSavedWorldSpawn(directory); ok {
		t.Fatal("invalid persisted spawn was accepted")
	}
}
