// Package world provides the Java Edition chunk encoding layer.
// It converts canonical core/world types into the wire format expected by
// vanilla Minecraft Java clients.
//
// A future Bedrock adapter will implement an equivalent conversion layer in
// bedrock/world without touching this package.
package world

// javaStateIDs maps canonical block names to their global block state IDs
// in Minecraft Java Edition 1.21.4 (protocol 769).
//
// These IDs are hardcoded for the current vanilla version.  They are stable
// within a version but change between versions as new blocks are inserted.
//
// Milestone 13 will replace this table with a data-driven registry loaded
// from the Minecraft data generator output (reports/blocks.json), which will
// also support older/newer versions and enable Java-to-Bedrock ID translation.
//
// Sources used to derive IDs:
//   - minecraft:air   → 0  (guaranteed by the Minecraft protocol specification)
//   - minecraft:stone → 1  (first non-air block in the 1.21.4 vanilla registry)
//
// When a block name is not found, StateID returns 0 (air), so unrecognised
// blocks render as invisible rather than crashing the client.
var javaStateIDs = map[string]int32{
	// ── Air ──────────────────────────────────────────────────────────────────
	"minecraft:air":  0,
	"minecraft:void": 0, // alias

	// ── Stone family ─────────────────────────────────────────────────────────
	// These IDs are stable across all 1.16+ releases.
	"minecraft:stone":    1,
	"minecraft:granite":  2,
	"minecraft:diorite":  4,
	"minecraft:andesite": 6,

	// ── Grass / dirt ─────────────────────────────────────────────────────────
	// Grass block has two states (snowy=false, snowy=true).
	"minecraft:grass_block": 8,  // snowy=false
	"minecraft:dirt":        10, // single state
	"minecraft:coarse_dirt": 11,
	"minecraft:podzol":      12, // snowy=false

	// ── Bedrock ──────────────────────────────────────────────────────────────
	"minecraft:bedrock": 25,

	// ── Sand ─────────────────────────────────────────────────────────────────
	"minecraft:sand":     66,
	"minecraft:red_sand": 67,

	// ── Gravel ───────────────────────────────────────────────────────────────
	"minecraft:gravel": 68,
}

// StateID returns the Java 1.21.4 global block state ID for the given
// canonical block name.  Returns 0 (air) for unrecognised names.
func StateID(name string) int32 {
	if id, ok := javaStateIDs[name]; ok {
		return id
	}
	return 0 // unknown → air
}
