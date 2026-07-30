package registry

import "GoCraft/java/network"

// VanillaProvider implements Provider using the Known Packs shortcut for
// vanilla Minecraft 1.21.4.
//
// It advertises "minecraft:core" at version "1.21.4".  A vanilla 1.21.4 client
// confirms it has this pack cached and skips downloading registry data for all
// standard registries (dimension types, biomes, damage types, etc.), instead
// using its bundled copy.
//
// Limitations (resolved in Milestone 13):
//
//   - Custom dimensions, biomes, or damage types require ExplicitProvider.
//   - Clients running different Minecraft versions may not recognise the pack.
//   - Bedrock-edition translation requires explicit registry IDs.
type VanillaProvider struct{}

// Packs returns the single "minecraft:core" pack for 1.21.4.
func (p *VanillaProvider) Packs() []Pack {
	return []Pack{{
		Namespace: "minecraft",
		ID:        "core",
		Version:   "1.21.4",
	}}
}

// SendRegistries sends no packets — the client uses its cached vanilla data.
func (p *VanillaProvider) SendRegistries(_ *network.ClientConn) error {
	return nil
}
