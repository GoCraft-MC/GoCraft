package config

import (
	"strings"
	"testing"
)

func TestWorldSeedEnvironmentOverride(t *testing.T) {
	t.Setenv("GOCRAFT_WORLD_SEED", "-9223372036854775808")
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if cfg.WorldSeed != -9223372036854775808 {
		t.Fatalf("WorldSeed = %d, want signed 64-bit override", cfg.WorldSeed)
	}
}

func TestWorldSeedEnvironmentRejectsInvalidValue(t *testing.T) {
	t.Setenv("GOCRAFT_WORLD_SEED", "not-a-seed")
	cfg := defaults()
	err := cfg.ApplyEnvOverrides()
	if err == nil || !strings.Contains(err.Error(), "GOCRAFT_WORLD_SEED") {
		t.Fatalf("ApplyEnvOverrides error = %v, want named seed error", err)
	}
}

func TestStreamingEnvironmentOverrides(t *testing.T) {
	t.Setenv("GOCRAFT_VIEW_DISTANCE", "10")
	t.Setenv("GOCRAFT_PREGENERATE_RADIUS", "16")
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if cfg.ViewDistance != 10 || cfg.PreGenerateRadius != 16 {
		t.Fatalf("streaming distances=(%d,%d), want (10,16)", cfg.ViewDistance, cfg.PreGenerateRadius)
	}
}

func TestPregenerationRadiusCannotBeSmallerThanView(t *testing.T) {
	cfg := defaults()
	cfg.ViewDistance = 12
	cfg.PreGenerateRadius = 8
	if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), "pregenerate_radius") {
		t.Fatalf("validate error=%v, want pregenerate_radius error", err)
	}
}
