package debuglog

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestConfigureKeepsFeaturesIndependent(t *testing.T) {
	t.Cleanup(func() { Configure(Settings{}) })

	Configure(Settings{
		EntityTickOverruns: true,
		BedrockChunks:      true,
	})

	if !Enabled(EntityTickOverruns) {
		t.Fatal("EntityTickOverruns is disabled, want enabled")
	}
	if !Enabled(BedrockChunks) {
		t.Fatal("BedrockChunks is disabled, want enabled")
	}
	for _, feature := range []Feature{
		StartupRegistry,
		WorldLoading,
		MobSpawning,
		Autosaves,
		EntityEvents,
		BedrockCatalogues,
		BedrockLogin,
		BedrockInventory,
		Profiling,
	} {
		if Enabled(feature) {
			t.Fatalf("feature %d is enabled, want disabled", feature)
		}
	}
}

func TestInfoEmitsOnlyWhenFeatureEnabled(t *testing.T) {
	previousLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
		Configure(Settings{})
	})

	var output bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))

	Configure(Settings{})
	Info(BedrockLogin, "hidden diagnostic")
	if output.Len() != 0 {
		t.Fatalf("disabled diagnostic output = %q, want empty", output.String())
	}

	Configure(Settings{BedrockLogin: true})
	Info(BedrockLogin, "visible diagnostic", "player", "Sushii4025")
	if got := output.String(); !strings.Contains(got, "visible diagnostic") || !strings.Contains(got, "Sushii4025") {
		t.Fatalf("enabled diagnostic output = %q", got)
	}
}
