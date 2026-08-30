// Package world provides the Java Edition chunk encoding layer.
// It converts canonical core/world types into the wire format expected by
// vanilla Minecraft Java clients.
//
// A future Bedrock adapter will implement an equivalent conversion layer in
// bedrock/world without touching this package.
package world

import coreworld "GoCraft/core/world"

// javaStateIDs maps canonical resource-location keys to their global block
// state IDs for Minecraft Java Edition 1.21.4 (protocol 769).
//
// Keys are either:
//
//	"namespace:name"                    — default state for this block
//	"namespace:name[k1=v1,k2=v2,…]"   — property-specific state
//
// Properties in compound keys are sorted alphabetically to match the output of
// core/world.Block.Key(), so two Block values with the same properties always
// resolve to the same Java ID regardless of insertion order.
//
// Populated at init time by registry.go from the embedded blocks.json.
var javaStateIDs map[string]int32

// blockTypeIDs maps block resource locations to registry IDs used by packets
// such as Block Event. These IDs are distinct from global block-state IDs.
var blockTypeIDs map[string]int32

// biomeIDs maps canonical biome resource locations to their protocol IDs for
// Minecraft Java Edition 1.21.4 (protocol 769).
// Populated at init time by registry.go from the embedded registries.json.
var biomeIDs map[string]int32

// blockEntityTypeIDs maps Java block entity resource locations to protocol IDs.
var blockEntityTypeIDs map[string]int32

// StateID returns the Java 1.21.4 global block state ID for the given canonical Block.
//
// Lookup order:
//  1. Full canonical key  "namespace:name[prop1=val1,…]"  — property-specific state ID.
//  2. Resource location   "namespace:name"                 — default state fallback.
//
// Logs a one-time warning for blocks not found in the registry, then returns 0
// (air) so unknown blocks render as invisible rather than crashing the client.
// HasExactState reports whether the complete property-bearing state exists in
// the embedded 1.21.4 report. It does not apply the default-state fallback.
func HasExactState(b coreworld.Block) bool {
	_, ok := javaStateIDs[b.Key()]
	return ok
}
func StateID(b coreworld.Block) int32 {
	if id, ok := javaStateIDs[b.Key()]; ok {
		return id
	}
	if id, ok := javaStateIDs[b.ResourceLocation()]; ok {
		return id
	}
	warnUnknown("minecraft:block", b.ResourceLocation())
	return 0
}

// BiomeID returns the Java 1.21.4 protocol biome ID for a canonical biome
// resource location (e.g. "minecraft:plains").
//
// Logs a one-time warning for biomes not found in the registry, then returns
// the ID for "minecraft:plains" (1) as a safe fallback.
// BlockEntityTypeID resolves a Java 1.21.4 block entity type.
func BlockEntityTypeID(name string) (int32, bool) {
	id, ok := blockEntityTypeIDs[name]
	return id, ok
}

// BlockTypeID resolves a block's minecraft:block registry ID.
func BlockTypeID(name string) (int32, bool) {
	id, ok := blockTypeIDs[name]
	return id, ok
}
func BiomeID(name string) int32 {
	if id, ok := biomeIDs[name]; ok {
		return id
	}
	warnUnknown("minecraft:worldgen/biome", name)
	return biomeIDs["minecraft:plains"] // safe fallback
}
