// Package config handles loading and saving server configuration from YAML.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// BedrockConfig holds configuration for the Bedrock Edition UDP listener.
type BedrockConfig struct {
	// Enabled controls whether the Bedrock listener is started at all.
	// Set to false to run a Java-only server.
	Enabled bool `yaml:"enabled"`

	// Address is the UDP listen address for RakNet connections.
	// The standard Bedrock port is 19132.
	Address string `yaml:"address"`

	// OnlineMode requires connecting players to authenticate with Xbox Live.
	// Disable only for LAN testing; unauthenticated XUIDs are NOT treated as
	// trusted global identities.
	OnlineMode bool `yaml:"online_mode"`
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

	// WorldDir is the path to the Minecraft world folder containing region/,
	// level.dat, etc.  Leave empty to disable Anvil persistence and run with
	// a freshly generated flat world on every startup.
	WorldDir string `yaml:"world_dir"`

	// Bedrock Edition UDP listener settings.
	Bedrock BedrockConfig `yaml:"bedrock"`
}

// defaults returns a Config populated with sane out-of-the-box values.
func defaults() *Config {
	return &Config{
		JavaEnabled:     true,
		Host:            "0.0.0.0",
		Port:            25565,
		MOTD:            "A GoCraft Server",
		MaxPlayers:      20,
		VersionName:     "1.21.4",
		ProtocolVersion: 769, // Minecraft Java Edition 1.21.4
		OnlineMode:      false,
		Bedrock: BedrockConfig{
			Enabled:    false,
			Address:    "0.0.0.0:19132",
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
	if c.MaxPlayers < 0 {
		return errors.New("max_players must be >= 0")
	}
	if c.Bedrock.Enabled && c.Bedrock.Address == "" {
		return errors.New("bedrock.address must not be empty when bedrock is enabled")
	}
	return nil
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
