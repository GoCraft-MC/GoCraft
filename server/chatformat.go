package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"GoCraft/java/handler"
	"gopkg.in/yaml.v3"
)

// chatFormatConfig holds the chat line template from configuration/chatformat.yml.
type chatFormatConfig struct {
	Format        string            `yaml:"format"`
	BedrockFormat string            `yaml:"bedrock_format"` // optional; falls back to Format with gradient/hex collapsed
	glyphs        map[string]string // loaded separately from configuration/glyphs.yml
}

const defaultChatFormat = "<gradient:#5865F2:#EB459E>{player}</gradient> <white>:</white> <gray>{message}</gray>"

const defaultChatFormatYAML = `# Chat format configuration for GoCraft.
# Supports MiniMessage tags: <red>, <gold>, <bold>, <#RRGGBB>,
# <gradient:#FF0000:#00FF00>, &c legacy codes, etc.
#
# Placeholders:
#   {prefix}  — the player's highest-weight group prefix (e.g. "[MOD] ")
#   {player}  — the player's username
#   {message} — the chat message (safe — cannot inject formatting tags)
format: "{prefix}<gradient:#5865F2:#EB459E>{player}</gradient> <white>:</white> <gray>{message}</gray>"

# Bedrock clients don't support hex colors or gradients in chat.
# This format is used for Bedrock players instead. Supports named colors only.
bedrock_format: "{prefix}<light_purple>{player}</light_purple> <white>:</white> <gray>{message}</gray>"
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

// apply substitutes placeholders and runs the result through the MiniMessage
// parser.  The player's message is escaped first so players cannot inject
// formatting tags — only the format template and prefix support MiniMessage.
func (cf *chatFormatConfig) apply(prefix, player, message string) string {
	s := cf.Format
	s = strings.ReplaceAll(s, "{prefix}", prefix)
	s = strings.ReplaceAll(s, "{player}", player)
	s = strings.ReplaceAll(s, "{message}", handler.EscapeMiniMessage(message))
	return handler.ParseMiniMessageWithOptions(s, handler.MMOptions{Glyphs: cf.glyphs})
}

// applyBedrock is like apply but produces a Bedrock-safe §-coded string.
// If bedrock_format is set in chatformat.yml it is used as the template;
// otherwise the standard format is run through the Bedrock-safe renderer
// which collapses gradients and hex colors to the nearest named §-color.
func (cf *chatFormatConfig) applyBedrock(prefix, player, message string) string {
	tmpl := cf.BedrockFormat
	if tmpl == "" {
		tmpl = cf.Format
	}
	s := tmpl
	s = strings.ReplaceAll(s, "{prefix}", prefix)
	s = strings.ReplaceAll(s, "{player}", player)
	s = strings.ReplaceAll(s, "{message}", handler.EscapeMiniMessage(message))
	return handler.ParseMiniMessageBedrock(s)
}
