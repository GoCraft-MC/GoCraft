// Package world provides the Java Edition chunk encoding layer.
// It converts canonical core/world types into the wire format expected by
// vanilla Minecraft Java clients.
//
// A future Bedrock adapter will implement an equivalent conversion layer in
// bedrock/world without touching this package.
package world

import coreworld "GoCraft/core/world"

// javaStateIDs maps canonical resource locations to their default global block
// state IDs in Minecraft Java Edition 1.21.4 (protocol 769).
//
// Keys are "namespace:name" strings (no properties).  For blocks whose default
// state is unambiguous (stone, bedrock, sand…) a single entry suffices.
// Blocks with property-dependent IDs (e.g. grass_block[snowy=false/true]) will
// be extended to a property-keyed sub-table in a later milestone.
//
// These IDs are hardcoded for the current vanilla version.  They are stable
// within a version but change between versions as new blocks are inserted.
//
// Milestone 13 will replace this table with a data-driven registry loaded from
// the Minecraft data generator output (reports/blocks.json).  At that point the
// table will also cover every property combination and support older/newer
// Java versions without touching any core/ types.
//
// When a block is not found, StateID returns 0 (air), so unrecognised blocks
// render as invisible rather than crashing the client.
var javaStateIDs = map[string]int32{
	// ── Air ──────────────────────────────────────────────────────────────────
	"minecraft:air":  0,
	"minecraft:void": 0, // alias

	// ── Stone family ─────────────────────────────────────────────────────────
	"minecraft:stone":    1,
	"minecraft:granite":  2,
	"minecraft:diorite":  4,
	"minecraft:andesite": 6,

	// ── Grass / dirt ─────────────────────────────────────────────────────────
	// Default state IDs (snowy=false, etc.).  Property variants handled later.
	"minecraft:grass_block": 8,  // default: snowy=false
	"minecraft:dirt":        10,
	"minecraft:coarse_dirt": 11,
	"minecraft:podzol":      12, // default: snowy=false

	// ── Bedrock ──────────────────────────────────────────────────────────────
	"minecraft:bedrock": 25,

	// ── Sand ─────────────────────────────────────────────────────────────────
	"minecraft:sand":     66,
	"minecraft:red_sand": 67,

	// ── Gravel ───────────────────────────────────────────────────────────────
	"minecraft:gravel": 68,
}

// StateID returns the Java 1.21.4 global block state ID for the given canonical Block.
//
// Lookup order:
//  1. Full canonical key  "namespace:name[prop1=val1,…]"  — property-specific state ID.
//     Properties are sorted alphabetically by Block.Key() so insertion order never
//     affects the result; two Block values with the same properties always resolve
//     to the same Java ID regardless of how the map was constructed.
//  2. Resource location   "namespace:name"                 — default state fallback.
//
// This lets javaStateIDs carry both default-state entries (most blocks) and
// property-specific overrides (e.g. "minecraft:grass_block[snowy=true]") without
// any special casing in the calling code.
//
// Returns 0 (air) for unrecognised blocks so unknown blocks render as invisible
// rather than crashing the client.
func StateID(b coreworld.Block) int32 {
	// Try the full canonical key first (property-specific override).
	if id, ok := javaStateIDs[b.Key()]; ok {
		return id
	}
	// Fall back to default state (resource location only).
	if id, ok := javaStateIDs[b.ResourceLocation()]; ok {
		return id
	}
	return 0 // unknown → air
}
