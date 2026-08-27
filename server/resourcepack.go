package server

import (
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// loadBedrockPack reads a Bedrock-format resource pack from a local file.
// Accepts .mcpack or .zip files that contain a manifest.json at their root.
func loadBedrockPack(path string) (*resource.Pack, error) {
	return resource.ReadPath(path)
}
