package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMobSpawningDefaultsReduceHostilePopulation(t *testing.T) {
	cfg := defaults()
	if cfg.MobSpawning.Hostile != 35 || cfg.MobSpawning.Passive != 16 {
		t.Fatalf("mob spawning defaults = %+v", cfg.MobSpawning)
	}
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMobSpawningPartialYAMLRetainsOtherDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yml")
	if err := os.WriteFile(path, []byte("mob_spawning:\n    hostile: 12\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MobSpawning.Hostile != 12 || cfg.MobSpawning.Passive != 16 || cfg.MobSpawning.WaterAmbient != 20 {
		t.Fatalf("partial mob spawning config = %+v", cfg.MobSpawning)
	}
}

func TestMobSpawningAllowsZeroAndRejectsUnsafeValues(t *testing.T) {
	cfg := defaults()
	cfg.MobSpawning.Hostile = 0
	if err := cfg.validate(); err != nil {
		t.Fatalf("zero should disable hostile natural spawning: %v", err)
	}
	cfg.MobSpawning.Hostile = -1
	if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), "mob_spawning.hostile") {
		t.Fatalf("negative hostile cap error = %v", err)
	}
	cfg = defaults()
	cfg.MobSpawning.WaterAmbient = 1001
	if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), "mob_spawning.water_ambient") {
		t.Fatalf("excessive water ambient cap error = %v", err)
	}
}
