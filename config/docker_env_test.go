package config

import (
	"strings"
	"testing"
)

func TestIdentityEnvironmentOverrides(t *testing.T) {
	t.Setenv("GOCRAFT_MOTD", "Mon serveur")
	t.Setenv("GOCRAFT_MAX_PLAYERS", "100")
	t.Setenv("GOCRAFT_VERSION_NAME", "1.21.5")
	t.Setenv("GOCRAFT_PROTOCOL_VERSION", "770")
	t.Setenv("GOCRAFT_VILLAGERS", "false")
	t.Setenv("GOCRAFT_DEFAULT_GAMEMODE", "creative")
	t.Setenv("GOCRAFT_DIFFICULTY", "hard")
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if cfg.MOTD != "Mon serveur" || cfg.MaxPlayers != 100 || cfg.VersionName != "1.21.5" ||
		cfg.ProtocolVersion != 770 || cfg.Villagers || cfg.DefaultGameMode != "creative" || cfg.Difficulty != "hard" {
		t.Fatalf("identity overrides not applied: %+v", cfg)
	}
}

func TestIdentityEnvironmentRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name, key, value, wantErr string
	}{
		{"protocol version", "GOCRAFT_PROTOCOL_VERSION", "not-a-number", "GOCRAFT_PROTOCOL_VERSION"},
		{"villagers flag", "GOCRAFT_VILLAGERS", "not-a-bool", "GOCRAFT_VILLAGERS"},
		{"gamemode", "GOCRAFT_DEFAULT_GAMEMODE", "hardcore", "default_gamemode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			cfg := defaults()
			err := cfg.ApplyEnvOverrides()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("ApplyEnvOverrides error = %v, want %s error", err, tc.wantErr)
			}
		})
	}
}

func TestPortAliasPrecedence(t *testing.T) {
	t.Setenv("GOCRAFT_HOST", "127.0.0.1")
	t.Setenv("GOCRAFT_PORT", "25566")
	t.Setenv("GOCRAFT_JAVA_HOST", "0.0.0.0")
	t.Setenv("GOCRAFT_JAVA_PORT", "25599")
	cfg := defaults()
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "0.0.0.0" || cfg.Port != 25599 {
		t.Fatalf("host:port = %s:%d, want the GOCRAFT_JAVA_* spelling to win", cfg.Host, cfg.Port)
	}
}

func TestBedrockAddressAlias(t *testing.T) {
	t.Setenv("GOCRAFT_BEDROCK_ADDRESS", "0.0.0.0:19132")
	cfg := defaults()
	cfg.Bedrock.Enabled = true
	if err := cfg.ApplyEnvOverrides(); err != nil {
		t.Fatal(err)
	}
	if cfg.Bedrock.Address != "0.0.0.0:19132" {
		t.Fatalf("bedrock.address = %q, want alias override", cfg.Bedrock.Address)
	}
}
