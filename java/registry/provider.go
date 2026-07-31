// Package registry manages the exchange of data-pack and registry information
// during the Minecraft Java Edition Configuration state.
//
// # Why this abstraction exists
//
// Minecraft 1.20.2 introduced the "Known Packs" optimisation: if the server
// declares a pack the client already has cached (e.g. "minecraft:core"), the
// client can skip downloading the full registry data for that pack.
// This is fast for vanilla clients but insufficient for:
//
//   - Custom dimensions and biomes (not in the vanilla pack).
//   - Older or modded clients that do not recognise the pack.
//   - Java-to-Bedrock translation, where the Bedrock adapter needs the full
//     canonical registry to map block/biome/entity IDs across editions.
//
// Provider decouples the configuration handler from the specific delivery
// strategy so that VanillaProvider (Known Packs shortcut) and a future
// ExplicitProvider (full registry data) can be swapped without touching the
// Configuration state machine.
package registry

import "GoCraft/java/network"

// Pack identifies a Minecraft data pack by its namespace, ID, and version.
// The server advertises packs it knows; the client replies with the subset
// it already has cached.
type Pack struct {
	Namespace string // "minecraft"
	ID        string // "core"
	Version   string // "1.21.4"
}

// Provider abstracts registry data delivery during the Configuration state.
//
// Two implementations are planned:
//
//   - VanillaProvider — Known Packs shortcut; sends ordered Registry Data keys
//     while omitting vanilla entry NBT for vanilla 1.21.4 clients.
//
//   - ExplicitProvider (future, Milestone 13) — advertises no packs and sends
//     full Registry Data for every required registry (dimension types, biomes,
//     damage types, etc.).  Required for custom content, non-vanilla clients,
//     and cross-edition translation.
type Provider interface {
	// Packs returns the packs to advertise in Clientbound Known Packs (0x0E S→C).
	// Returning an empty slice forces the client to download explicit registry data.
	Packs() []Pack

	// SendRegistries sends Registry Data packets (0x07 S→C) after the client
	// responds to Known Packs. Entries covered by selected packs may omit NBT,
	// but their registry and ordered keys must still be sent to assign IDs.
	SendRegistries(conn *network.ClientConn, selected []Pack) error

	// SendTags sends Update Tags (0x0D S→C) using IDs assigned by the
	// synchronized and built-in registries for this protocol version.
	SendTags(conn *network.ClientConn) error

	// DimensionTypeID returns the ID assigned to name by the dimension_type
	// registry sent for this provider during Configuration.
	DimensionTypeID(name string) (int32, error)
}
