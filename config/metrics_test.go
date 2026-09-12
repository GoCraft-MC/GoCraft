package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetricsDefaultsAreSafeAndPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Metrics.Enabled || cfg.Metrics.Address != "127.0.0.1:9225" {
		t.Fatalf("metrics defaults = %+v", cfg.Metrics)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "metrics:\n    enabled: false\n    address: 127.0.0.1:9225") {
		t.Fatal("generated configuration does not persist safe metrics defaults")
	}
}

func TestMetricsAddressValidation(t *testing.T) {
	for _, tc := range []struct {
		address string
		valid   bool
	}{
		{"127.0.0.1:9225", true}, {"localhost:1", true}, {"0.0.0.0:65535", true},
		{"[::1]:9225", true}, {":9225", true}, {"  [::1]:9225\t", true},
		{"", false}, {"localhost", false}, {"127.0.0.1:0", false},
		{"localhost:-1", false}, {"localhost:65536", false}, {"localhost:http", false},
		{"::1:9225", false}, {"localhost:", false}, {"localhost:1.5", false},
	} {
		t.Run(tc.address, func(t *testing.T) {
			cfg := defaults()
			cfg.Metrics = MetricsConfig{Enabled: true, Address: tc.address}
			err := cfg.validate()
			if (err == nil) != tc.valid {
				t.Fatalf("validate(%q) = %v, valid = %v", tc.address, err, tc.valid)
			}
			if cfg.Metrics.Address != strings.TrimSpace(tc.address) {
				t.Fatalf("address was not trimmed: %q", cfg.Metrics.Address)
			}
		})
	}
}

func TestMetricsEnvironmentOverrides(t *testing.T) {
	t.Setenv("GOCRAFT_METRICS_ENABLED", "true")
	t.Setenv("GOCRAFT_METRICS_ADDR", "[::1]:9226")
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if !cfg.Metrics.Enabled || cfg.Metrics.Address != "[::1]:9226" {
		t.Fatalf("metrics overrides = %+v", cfg.Metrics)
	}
	t.Setenv("GOCRAFT_METRICS_ENABLED", "invalid")
	if err := cfg.ApplyEnvOverrides(); err == nil || !strings.Contains(err.Error(), "GOCRAFT_METRICS_ENABLED") {
		t.Fatalf("invalid boolean override = %v", err)
	}
	t.Setenv("GOCRAFT_METRICS_ENABLED", "true")
	t.Setenv("GOCRAFT_METRICS_ADDR", "localhost:0")
	if err := cfg.ApplyEnvOverrides(); err == nil || !strings.Contains(err.Error(), "metrics.address") {
		t.Fatalf("invalid enabled address = %v", err)
	}
	t.Setenv("GOCRAFT_METRICS_ENABLED", "false")
	if err := cfg.ApplyEnvOverrides(); err != nil || cfg.Metrics.Enabled {
		t.Fatalf("disabled metrics rejected an unused address: %v", err)
	}
}
