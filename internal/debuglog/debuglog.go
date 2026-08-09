// Package debuglog provides feature-scoped diagnostic logging switches.
package debuglog

import (
	"log/slog"
	"sync/atomic"
)

// Feature identifies one independently configurable diagnostic log category.
type Feature uint8

const (
	StartupRegistry Feature = iota
	WorldLoading
	MobSpawning
	Autosaves
	EntityEvents
	EntityTickOverruns
	BedrockCatalogues
	BedrockLogin
	BedrockChunks
	BedrockInventory
	Profiling
)

// Settings contains process-wide diagnostic logging choices.
type Settings struct {
	StartupRegistry    bool
	WorldLoading       bool
	MobSpawning        bool
	Autosaves          bool
	EntityEvents       bool
	EntityTickOverruns bool
	BedrockCatalogues  bool
	BedrockLogin       bool
	BedrockChunks      bool
	BedrockInventory   bool
	Profiling          bool
}

var enabledMask atomic.Uint64

// Configure atomically replaces all diagnostic logging switches.
func Configure(settings Settings) {
	var mask uint64
	set := func(feature Feature, enabled bool) {
		if enabled {
			mask |= uint64(1) << feature
		}
	}
	set(StartupRegistry, settings.StartupRegistry)
	set(WorldLoading, settings.WorldLoading)
	set(MobSpawning, settings.MobSpawning)
	set(Autosaves, settings.Autosaves)
	set(EntityEvents, settings.EntityEvents)
	set(EntityTickOverruns, settings.EntityTickOverruns)
	set(BedrockCatalogues, settings.BedrockCatalogues)
	set(BedrockLogin, settings.BedrockLogin)
	set(BedrockChunks, settings.BedrockChunks)
	set(BedrockInventory, settings.BedrockInventory)
	set(Profiling, settings.Profiling)
	enabledMask.Store(mask)
}

// Enabled reports whether a diagnostic category is currently enabled.
func Enabled(feature Feature) bool {
	return enabledMask.Load()&(uint64(1)<<feature) != 0
}

// Info emits an informational diagnostic line only when feature is enabled.
func Info(feature Feature, message string, args ...any) {
	if Enabled(feature) {
		slog.Info(message, args...)
	}
}
