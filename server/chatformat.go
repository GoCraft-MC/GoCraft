package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// chatFormatConfig holds the chat line template from configuration/chatformat.yml.
type chatFormatConfig struct {
	Format string `yaml:"format"`
}

const defaultChatFormat = "<{player}> {message}"

const defaultChatFormatYAML = `# Chat format configuration for GoCraft.
# Placeholders:
#   {prefix}  — the player's highest-weight group prefix (e.g. "[MOD] ")
#   {player}  — the player's username
#   {message} — the chat message
format: "{prefix}<{player}> {message}"
`

// loadChatFormat reads the file at path.  If it does not exist the
// configuration/ directory and a default chatformat.yml are created
// automatically so admins can see and edit the file on Pterodactyl.
func loadChatFormat(path string) (*chatFormatConfig, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cf := &chatFormatConfig{Format: defaultChatFormat}
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr == nil {
			_ = os.WriteFile(path, []byte(defaultChatFormatYAML), 0o644)
		}
		return cf, nil
	}
	if err != nil {
		return nil, err
	}
	var cf chatFormatConfig
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, err
	}
	if cf.Format == "" {
		cf.Format = defaultChatFormat
	}
	return &cf, nil
}

// apply substitutes {prefix}, {player}, and {message} placeholders.
func (cf *chatFormatConfig) apply(prefix, player, message string) string {
	s := cf.Format
	s = strings.ReplaceAll(s, "{prefix}", prefix)
	s = strings.ReplaceAll(s, "{player}", player)
	s = strings.ReplaceAll(s, "{message}", message)
	return s
}
