package server

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type glyphsFile struct {
	Glyphs map[string]string `yaml:"glyphs"`
}

const defaultGlyphsYAML = `# Glyph configuration for GoCraft.
# Maps glyph names to Unicode characters for use in chat prefixes and format strings.
# Use <glyph:name> inside MiniMessage templates (chatformat.yml, group prefixes, etc.)
#
# These characters are rendered as images by your resource pack.
# Private Use Area characters (U+E000–U+F8FF) are the standard choice.
#
# Example usage in chatformat.yml:
#   prefix: "<red><glyph:crown> [OWNER]</red> "
#
# Example usage in-game:
#   /gocraft group owner setprefix <glyph:logo> [OWNER]
glyphs:
  logo:  "\uE000"
  crown: "\uE001"
  star:  "\uE002"
  heart: "\uE003"
  sword: "\uE004"
`

// loadGlyphs reads configuration/glyphs.yml.  If missing it creates the file
// with defaults and returns an empty map (no glyphs configured yet).
func loadGlyphs(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr == nil {
			_ = os.WriteFile(path, []byte(defaultGlyphsYAML), 0o644)
		}
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f glyphsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Glyphs == nil {
		return map[string]string{}, nil
	}
	return f.Glyphs, nil
}
