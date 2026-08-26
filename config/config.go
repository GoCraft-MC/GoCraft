// Package config handles loading and saving server configuration from YAML.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// BedrockConfig holds configuration for the Bedrock Edition UDP listener.
type BedrockConfig struct {
	// Enabled controls whether the Bedrock listener is started at all.
	// Set to false to run a Java-only server.
	Enabled bool `yaml:"enabled"`

	// Address is the UDP listen address for RakNet connections.
	// GoCraft defaults to the deployment's requested Bedrock UDP port 19106.
	Address string `yaml:"address"`

	// OnlineMode requires connecting players to authenticate with Xbox Live.
	// Disable only for LAN testing; unauthenticated XUIDs are NOT treated as
	// trusted global identities.
	OnlineMode bool `yaml:"online_mode"`
}

const (
	WorldStorageDisk   = "disk"
	WorldStorageMemory = "memory"
)

// CombatConfig controls Java combat timing and knockback. Disabling the attack
// cooldown with the default values provides a legacy 1.8-style feel.
type CombatConfig struct {
	AttackCooldown      bool    `yaml:"attack_cooldown"`
	KnockbackHorizontal float64 `yaml:"knockback_horizontal"`
	KnockbackVertical   float64 `yaml:"knockback_vertical"`
}

// ItemTooltipConfig controls the extra information GoCraft adds to item
// tooltips. Vanilla attributes can be hidden independently because a legacy
// no-cooldown attack-speed override would otherwise appear as an ugly 1024.
type ItemTooltipConfig struct {
	ShowDurability        bool `yaml:"show_durability"`
	ShowAttributes        bool `yaml:"show_attributes"`
	HideVanillaAttributes bool `yaml:"hide_vanilla_attributes"`
}

type ClearLagTargets struct {
	DroppedItems   bool `yaml:"dropped_items"`
	ExperienceOrbs bool `yaml:"experience_orbs"`
	Projectiles    bool `yaml:"projectiles"`
	PrimedTNT      bool `yaml:"primed_tnt"`
	FallingBlocks  bool `yaml:"falling_blocks"`
	Boats          bool `yaml:"boats"`
	PassiveMobs    bool `yaml:"passive_mobs"`
	HostileMobs    bool `yaml:"hostile_mobs"`
}

type ClearLagConfig struct {
	Enabled                 bool            `yaml:"enabled"`
	IntervalSeconds         int             `yaml:"interval_seconds"`
	MinimumEntityAgeSeconds int             `yaml:"minimum_entity_age_seconds"`
	WarningSeconds          []int           `yaml:"warning_seconds"`
	WarningMessage          string          `yaml:"warning_message"`
	CompleteMessage         string          `yaml:"complete_message"`
	Remove                  ClearLagTargets `yaml:"remove"`
}

// WhitelistConfig controls the shared Java/Bedrock player whitelist. Names are
// compared case-insensitively and persisted to whitelist.json at runtime.
type WhitelistConfig struct {
	Enabled bool     `yaml:"enabled"`
	Players []string `yaml:"players"`
}

type PermissionEditorConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Address        string `yaml:"address"`
	PublicURL      string `yaml:"public_url"`
	SessionMinutes int    `yaml:"session_minutes"`
}

// DebugConfig controls verbose diagnostic log categories. All switches default
// to false so routine console and latest.log output remain concise.
type DebugConfig struct {
	StartupRegistry      bool `yaml:"startup_registry"`
	EnvironmentOverrides bool `yaml:"environment_overrides"`
	WorldLoading         bool `yaml:"world_loading"`
	MobSpawning          bool `yaml:"mob_spawning"`
	Autosaves            bool `yaml:"autosaves"`
	EntityEvents         bool `yaml:"entity_events"`
	EntityTickOverruns   bool `yaml:"entity_tick_overruns"`
	BedrockCatalogues    bool `yaml:"bedrock_catalogues"`
	BedrockLogin         bool `yaml:"bedrock_login"`
	BedrockChunks        bool `yaml:"bedrock_chunks"`
	BedrockInventory     bool `yaml:"bedrock_inventory"`
	Profiling            bool `yaml:"profiling"`
}

// MobSpawningConfig controls the natural-mob caps for one complete 17x17
// chunk simulation area. Smaller loaded areas scale these values down, and 0
// disables natural spawning for that category.
type MobSpawningConfig struct {
	Hostile                   int `yaml:"hostile"`
	Passive                   int `yaml:"passive"`
	Ambient                   int `yaml:"ambient"`
	Axolotls                  int `yaml:"axolotls"`
	UndergroundWaterCreatures int `yaml:"underground_water_creatures"`
	WaterCreatures            int `yaml:"water_creatures"`
	WaterAmbient              int `yaml:"water_ambient"`
}

// Config holds all server configuration values.
type Config struct {
	// JavaEnabled controls whether the Java Edition TCP listener is started.
	// Set to false to run a Bedrock-only server.
	JavaEnabled bool `yaml:"java_enabled"`

	// Network (Java Edition)
	Host string `yaml:"host"`
	Port int    `yaml:"port"`

	// Identity
	MOTD            string `yaml:"motd"`
	MaxPlayers      int    `yaml:"max_players"`
	VersionName     string `yaml:"version_name"`
	ProtocolVersion int    `yaml:"protocol_version"`

	// Behaviour (Java Edition)
	OnlineMode bool `yaml:"online_mode"`

	// Villagers controls whether villager entities are spawned in villages.
	// Set to false to disable NPC spawning (structures still generate).
	Villagers bool `yaml:"villagers"`

	// WorldStorage selects "disk" (Anvil persistence plus an in-memory cache)
	// or "memory" (no disk reads or writes).
	WorldStorage string `yaml:"world_storage"`

	// WorldDir is the Anvil world folder used when WorldStorage is "disk".
	WorldDir string `yaml:"world_dir"`

	// WorldSeed controls deterministic overworld terrain generation for chunks
	// absent from Anvil storage. The same seed always produces the same terrain.
	WorldSeed int64 `yaml:"world_seed"`

	// ViewDistance is the radius, in chunks, sent to Java clients. PreGenerateRadius
	// warms a larger square in the background so movement rarely waits on worldgen.
	ViewDistance      int `yaml:"view_distance"`
	PreGenerateRadius int `yaml:"pregenerate_radius"`

	// MaxCachedChunks bounds clean chunks retained in RAM. Dirty memory-mode
	// chunks remain pinned so unsaved changes are never silently discarded.
	MaxCachedChunks int `yaml:"max_cached_chunks"`

	// Difficulty controls hostile-mob behaviour and spawning.
	// Valid values: peaceful, easy, normal, hard.  Default: normal.
	// On peaceful no hostile mobs are spawned and existing ones are removed.
	Difficulty string `yaml:"difficulty"`

	DefaultGameMode string            `yaml:"default_gamemode"`
	MobSpawning     MobSpawningConfig `yaml:"mob_spawning"`

	// Operators bootstraps named operators. Runtime /op changes are persisted
	// separately in ops.json, matching the vanilla server convention.
	Operators        []string
	Whitelist        WhitelistConfig        `yaml:"whitelist"`
	PermissionEditor PermissionEditorConfig `yaml:"permission_editor"`
	Debug            DebugConfig            `yaml:"debug"`

	// Combat timing and knockback settings.
	Combat CombatConfig `yaml:"combat"`

	ItemTooltips ItemTooltipConfig `yaml:"item_tooltips"`

	ClearLag ClearLagConfig `yaml:"clear_lag"`

	// Bedrock Edition UDP listener settings.
	Bedrock BedrockConfig `yaml:"bedrock"`
}

// defaults returns a Config populated with sane out-of-the-box values.
func defaults() *Config {
	return &Config{
		JavaEnabled:       true,
		Host:              "0.0.0.0",
		Port:              25565,
		MOTD:              "A GoCraft Server",
		MaxPlayers:        20,
		VersionName:       "1.21.4",
		ProtocolVersion:   769, // Minecraft Java Edition 1.21.4
		OnlineMode:        false,
		Villagers:         true,
		WorldStorage:      WorldStorageDisk,
		WorldDir:          "world",
		ViewDistance:      8,
		PreGenerateRadius: 8,
		MaxCachedChunks:   256,
		Difficulty:        "normal",
		DefaultGameMode:   "survival",
		MobSpawning: MobSpawningConfig{
			Hostile:                   35,
			Passive:                   16,
			Ambient:                   15,
			Axolotls:                  5,
			UndergroundWaterCreatures: 5,
			WaterCreatures:            5,
			WaterAmbient:              20,
		},
		Whitelist: WhitelistConfig{Enabled: false, Players: []string{}},
		PermissionEditor: PermissionEditorConfig{
			Enabled: true, Address: "127.0.0.1:8080",
			PublicURL: "http://127.0.0.1:8080", SessionMinutes: 15,
		},
		Combat: CombatConfig{
			AttackCooldown:      false,
			KnockbackHorizontal: 0.4,
			KnockbackVertical:   0.4,
		},
		ItemTooltips: ItemTooltipConfig{
			ShowDurability:        true,
			ShowAttributes:        true,
			HideVanillaAttributes: true,
		},
		ClearLag: ClearLagConfig{
			Enabled:                 false,
			IntervalSeconds:         300,
			MinimumEntityAgeSeconds: 30,
			WarningSeconds:          []int{60, 30, 10, 5, 4, 3, 2, 1},
			WarningMessage:          "[ClearLag] Removing old entities in {seconds}s",
			CompleteMessage:         "[ClearLag] Removed {count} old entities",
			Remove: ClearLagTargets{
				DroppedItems:   true,
				ExperienceOrbs: true,
				Projectiles:    true,
				PrimedTNT:      true,
				FallingBlocks:  true,
			},
		},
		Bedrock: BedrockConfig{
			Enabled:    false,
			Address:    "0.0.0.0:19106",
			OnlineMode: true,
		},
	}
}

// Load reads configuration from the YAML file at path.
// If the file does not exist, default values are written to path and returned.
func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err2 := save(path, cfg); err2 != nil {
			return nil, fmt.Errorf("config: writing defaults to %s: %w", path, err2)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: invalid values in %s: %w", path, err)
	}

	return cfg, nil
}

// Addr returns the "host:port" string suitable for net.Listen.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// validate returns an error if required fields are out of range.
func (c *Config) validate() error {
	if c.JavaEnabled {
		if c.Port < 1 || c.Port > 65535 {
			return fmt.Errorf("port %d is out of range 1-65535", c.Port)
		}
		if c.ProtocolVersion <= 0 {
			return errors.New("protocol_version must be > 0")
		}
	}
	c.WorldStorage = strings.ToLower(strings.TrimSpace(c.WorldStorage))
	switch c.WorldStorage {
	case WorldStorageDisk:
		if strings.TrimSpace(c.WorldDir) == "" {
			return errors.New("world_dir must not be empty when world_storage is disk")
		}
	case WorldStorageMemory:
	default:
		return fmt.Errorf("world_storage %q must be disk or memory", c.WorldStorage)
	}
	if c.MaxPlayers < 0 {
		return errors.New("max_players must be >= 0")
	}
	if c.ViewDistance < 2 || c.ViewDistance > 32 {
		return fmt.Errorf("view_distance %d is out of range 2-32", c.ViewDistance)
	}
	if c.PreGenerateRadius < c.ViewDistance || c.PreGenerateRadius > 64 {
		return fmt.Errorf("pregenerate_radius %d must be between view_distance (%d) and 64", c.PreGenerateRadius, c.ViewDistance)
	}
	if c.MaxCachedChunks < 128 || c.MaxCachedChunks > 65536 {
		return fmt.Errorf("max_cached_chunks %d must be between 128 and 65536", c.MaxCachedChunks)
	}
	c.Difficulty = strings.ToLower(strings.TrimSpace(c.Difficulty))
	switch c.Difficulty {
	case "peaceful", "easy", "normal", "hard":
	default:
		return fmt.Errorf("difficulty %q must be peaceful, easy, normal, or hard", c.Difficulty)
	}
	c.DefaultGameMode = strings.ToLower(strings.TrimSpace(c.DefaultGameMode))
	switch c.DefaultGameMode {
	case "survival", "creative", "adventure", "spectator":
	default:
		return fmt.Errorf("default_gamemode %q must be survival, creative, adventure, or spectator", c.DefaultGameMode)
	}
	for name, limit := range map[string]int{
		"hostile":                     c.MobSpawning.Hostile,
		"passive":                     c.MobSpawning.Passive,
		"ambient":                     c.MobSpawning.Ambient,
		"axolotls":                    c.MobSpawning.Axolotls,
		"underground_water_creatures": c.MobSpawning.UndergroundWaterCreatures,
		"water_creatures":             c.MobSpawning.WaterCreatures,
		"water_ambient":               c.MobSpawning.WaterAmbient,
	} {
		if limit < 0 || limit > 1000 {
			return fmt.Errorf("mob_spawning.%s %d must be between 0 and 1000", name, limit)
		}
	}
	if c.Combat.KnockbackHorizontal < 0 || c.Combat.KnockbackHorizontal > 4 {
		return fmt.Errorf("combat.knockback_horizontal %.3f must be between 0 and 4", c.Combat.KnockbackHorizontal)
	}
	if c.Combat.KnockbackVertical < 0 || c.Combat.KnockbackVertical > 4 {
		return fmt.Errorf("combat.knockback_vertical %.3f must be between 0 and 4", c.Combat.KnockbackVertical)
	}
	if c.ClearLag.IntervalSeconds < 10 {
		return errors.New("clear_lag.interval_seconds must be at least 10")
	}
	if c.ClearLag.MinimumEntityAgeSeconds < 0 {
		return errors.New("clear_lag.minimum_entity_age_seconds must be >= 0")
	}
	for _, seconds := range c.ClearLag.WarningSeconds {
		if seconds <= 0 || seconds >= c.ClearLag.IntervalSeconds {
			return fmt.Errorf("clear_lag warning %d must be between 1 and interval_seconds-1", seconds)
		}
	}
	if c.Bedrock.Enabled && c.Bedrock.Address == "" {
		return errors.New("bedrock.address must not be empty when bedrock is enabled")
	}
	if c.PermissionEditor.Enabled {
		if _, _, err := net.SplitHostPort(c.PermissionEditor.Address); err != nil {
			return fmt.Errorf("permission_editor.address: %w", err)
		}
		parsed, err := url.ParseRequestURI(c.PermissionEditor.PublicURL)
		if err != nil || parsed.Host == "" || parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("permission_editor.public_url must be an absolute HTTP(S) URL")
		}
		if c.PermissionEditor.SessionMinutes < 1 || c.PermissionEditor.SessionMinutes > 1440 {
			return errors.New("permission_editor.session_minutes must be between 1 and 1440")
		}
	}
	return nil
}

// ApplyEnvOverrides reads well-known environment variables and overrides any
// YAML values that are present.  Call this after Load() and before passing
// Config to the server.  Validation is re-run after applying overrides so
// an invalid combination (e.g. GOCRAFT_JAVA_PORT=0) is caught early.
//
// Environment variables (all optional; empty = no override):
//
//	GOCRAFT_JAVA_HOST         Java TCP bind host          (default: 0.0.0.0)
//	GOCRAFT_JAVA_PORT         Java TCP port number        (default: 25565)
//	GOCRAFT_JAVA_ENABLED      "true"/"false"              (default: true)
//	GOCRAFT_ONLINE_MODE       Java auth required          (default: false)
//	GOCRAFT_MOTD              Server MOTD string
//	GOCRAFT_MAX_PLAYERS       Max concurrent players
//	GOCRAFT_WORLD_STORAGE     disk or memory              (default: disk)
//	GOCRAFT_WORLD_DIR         Anvil world directory path
//	GOCRAFT_WORLD_SEED        Signed 64-bit terrain seed
//	GOCRAFT_VIEW_DISTANCE     Java chunk view radius        (default: 8)
//	GOCRAFT_PREGENERATE_RADIUS Background generation radius (default: 12)
//	GOCRAFT_MAX_CACHED_CHUNKS Clean chunk cache limit       (default: 768)
//	GOCRAFT_DIFFICULTY        peaceful/easy/normal/hard    (default: normal)
//	GOCRAFT_WHITELIST_ENABLED "true"/"false"              (default: false)
//	GOCRAFT_BEDROCK_ENABLED   "true"/"false"              (default: false)
//	GOCRAFT_BEDROCK_ADDR      Bedrock UDP address         (default: 0.0.0.0:19106)
//	GOCRAFT_BEDROCK_ONLINE_MODE Xbox Live auth required   (default: true)
//	GOCRAFT_PERMISSION_EDITOR_ENABLED Enable browser permission editor
//	GOCRAFT_PERMISSION_EDITOR_ADDR    Editor HTTP bind address
//	GOCRAFT_PERMISSION_EDITOR_URL     Externally reachable editor base URL
func (c *Config) ApplyEnvOverrides() error {
	if v := os.Getenv("GOCRAFT_JAVA_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("GOCRAFT_JAVA_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_JAVA_PORT %q: %w", v, err)
		}
		c.Port = n
	}
	if v := os.Getenv("GOCRAFT_JAVA_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_JAVA_ENABLED %q: %w", v, err)
		}
		c.JavaEnabled = b
	}
	if v := os.Getenv("GOCRAFT_ONLINE_MODE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_ONLINE_MODE %q: %w", v, err)
		}
		c.OnlineMode = b
	}
	if v := os.Getenv("GOCRAFT_MOTD"); v != "" {
		c.MOTD = v
	}
	if v := os.Getenv("GOCRAFT_MAX_PLAYERS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_MAX_PLAYERS %q: %w", v, err)
		}
		c.MaxPlayers = n
	}
	if v := os.Getenv("GOCRAFT_WORLD_STORAGE"); v != "" {
		c.WorldStorage = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("GOCRAFT_WORLD_DIR"); v != "" {
		c.WorldDir = v
	}
	if v := os.Getenv("GOCRAFT_WORLD_SEED"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("GOCRAFT_WORLD_SEED %q: %w", v, err)
		}
		c.WorldSeed = n
	}
	if v := os.Getenv("GOCRAFT_VIEW_DISTANCE"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_VIEW_DISTANCE %q: %w", v, err)
		}
		c.ViewDistance = n
	}
	if v := os.Getenv("GOCRAFT_PREGENERATE_RADIUS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_PREGENERATE_RADIUS %q: %w", v, err)
		}
		c.PreGenerateRadius = n
	}
	if v := os.Getenv("GOCRAFT_MAX_CACHED_CHUNKS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_MAX_CACHED_CHUNKS %q: %w", v, err)
		}
		c.MaxCachedChunks = n
	}
	if v := os.Getenv("GOCRAFT_DIFFICULTY"); v != "" {
		c.Difficulty = strings.ToLower(strings.TrimSpace(v))
	}
	if v := os.Getenv("GOCRAFT_WHITELIST_ENABLED"); v != "" {
		value, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_WHITELIST_ENABLED %q: %w", v, err)
		}
		c.Whitelist.Enabled = value
	}
	if v := os.Getenv("GOCRAFT_ATTACK_COOLDOWN"); v != "" {
		value, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_ATTACK_COOLDOWN %q: %w", v, err)
		}
		c.Combat.AttackCooldown = value
	}
	if v := os.Getenv("GOCRAFT_KNOCKBACK_HORIZONTAL"); v != "" {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("GOCRAFT_KNOCKBACK_HORIZONTAL %q: %w", v, err)
		}
		c.Combat.KnockbackHorizontal = value
	}
	if v := os.Getenv("GOCRAFT_KNOCKBACK_VERTICAL"); v != "" {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("GOCRAFT_KNOCKBACK_VERTICAL %q: %w", v, err)
		}
		c.Combat.KnockbackVertical = value
	}
	if v := os.Getenv("GOCRAFT_BEDROCK_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_BEDROCK_ENABLED %q: %w", v, err)
		}
		c.Bedrock.Enabled = b
	}
	if v := os.Getenv("GOCRAFT_BEDROCK_ADDR"); v != "" {
		c.Bedrock.Address = v
	}
	if v := os.Getenv("GOCRAFT_BEDROCK_ONLINE_MODE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_BEDROCK_ONLINE_MODE %q: %w", v, err)
		}
		c.Bedrock.OnlineMode = b
	}
	if v := os.Getenv("GOCRAFT_PERMISSION_EDITOR_ENABLED"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("GOCRAFT_PERMISSION_EDITOR_ENABLED %q: %w", v, err)
		}
		c.PermissionEditor.Enabled = enabled
	}
	if v := os.Getenv("GOCRAFT_PERMISSION_EDITOR_ADDR"); v != "" {
		c.PermissionEditor.Address = v
	}
	if v := os.Getenv("GOCRAFT_PERMISSION_EDITOR_URL"); v != "" {
		c.PermissionEditor.PublicURL = strings.TrimRight(v, "/")
	}

	if c.Debug.EnvironmentOverrides {
		logEnvOverrides()
	}

	// Re-validate after overrides — env vars can produce invalid combinations.
	return c.validate()
}

// logEnvOverrides logs which values were overridden by environment variables.
func logEnvOverrides() {
	vars := []struct{ key, val string }{
		{"GOCRAFT_JAVA_HOST", os.Getenv("GOCRAFT_JAVA_HOST")},
		{"GOCRAFT_JAVA_PORT", os.Getenv("GOCRAFT_JAVA_PORT")},
		{"GOCRAFT_JAVA_ENABLED", os.Getenv("GOCRAFT_JAVA_ENABLED")},
		{"GOCRAFT_ONLINE_MODE", os.Getenv("GOCRAFT_ONLINE_MODE")},
		{"GOCRAFT_MOTD", os.Getenv("GOCRAFT_MOTD")},
		{"GOCRAFT_MAX_PLAYERS", os.Getenv("GOCRAFT_MAX_PLAYERS")},
		{"GOCRAFT_WORLD_STORAGE", os.Getenv("GOCRAFT_WORLD_STORAGE")},
		{"GOCRAFT_WORLD_DIR", os.Getenv("GOCRAFT_WORLD_DIR")},
		{"GOCRAFT_WORLD_SEED", os.Getenv("GOCRAFT_WORLD_SEED")},
		{"GOCRAFT_VIEW_DISTANCE", os.Getenv("GOCRAFT_VIEW_DISTANCE")},
		{"GOCRAFT_PREGENERATE_RADIUS", os.Getenv("GOCRAFT_PREGENERATE_RADIUS")},
		{"GOCRAFT_MAX_CACHED_CHUNKS", os.Getenv("GOCRAFT_MAX_CACHED_CHUNKS")},
		{"GOCRAFT_DIFFICULTY", os.Getenv("GOCRAFT_DIFFICULTY")},
		{"GOCRAFT_WHITELIST_ENABLED", os.Getenv("GOCRAFT_WHITELIST_ENABLED")},
		{"GOCRAFT_ATTACK_COOLDOWN", os.Getenv("GOCRAFT_ATTACK_COOLDOWN")},
		{"GOCRAFT_KNOCKBACK_HORIZONTAL", os.Getenv("GOCRAFT_KNOCKBACK_HORIZONTAL")},
		{"GOCRAFT_KNOCKBACK_VERTICAL", os.Getenv("GOCRAFT_KNOCKBACK_VERTICAL")},
		{"GOCRAFT_BEDROCK_ENABLED", os.Getenv("GOCRAFT_BEDROCK_ENABLED")},
		{"GOCRAFT_BEDROCK_ADDR", os.Getenv("GOCRAFT_BEDROCK_ADDR")},
		{"GOCRAFT_PERMISSION_EDITOR_ENABLED", os.Getenv("GOCRAFT_PERMISSION_EDITOR_ENABLED")},
		{"GOCRAFT_PERMISSION_EDITOR_ADDR", os.Getenv("GOCRAFT_PERMISSION_EDITOR_ADDR")},
		{"GOCRAFT_PERMISSION_EDITOR_URL", os.Getenv("GOCRAFT_PERMISSION_EDITOR_URL")},
		{"GOCRAFT_BEDROCK_ONLINE_MODE", os.Getenv("GOCRAFT_BEDROCK_ONLINE_MODE")},
	}
	for _, v := range vars {
		if v.val != "" {
			slog.Info("config: env override applied", "var", v.key, "value", v.val)
		}
	}
}

// save marshals cfg to YAML and writes it to path, creating parent directories.
func save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
