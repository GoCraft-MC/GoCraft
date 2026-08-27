package server

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// chatFormatConfig holds the chat line template from configuration/chatformat.yml.
type chatFormatConfig struct {
	Format string `yaml:"format"`
}

const defaultChatFormat = "<{player}> {message}"

// loadChatFormat reads configuration/chatformat.yml relative to the working
// directory.  Returns the default format when the file does not exist.
func loadChatFormat(path string) (*chatFormatConfig, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &chatFormatConfig{Format: defaultChatFormat}, nil
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
