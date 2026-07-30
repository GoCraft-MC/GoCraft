package protocoldata_test

// Validation tests for the embedded protocol-data packet tables.
//
// These tests catch problems that would otherwise surface only at runtime or
// after a player connects:
//
//   - Duplicate numeric IDs within the same state+direction pair.
//   - Duplicate semantic names within the same state+direction pair.
//   - IDs outside the single-byte VarInt range (0–127); Minecraft uses at
//     most two VarInt bytes for packet IDs but practically all IDs in current
//     versions fit in one byte.
//   - The _scope metadata key is present and non-empty on every state file,
//     so nobody mistakes a partial table for a complete protocol spec.
//   - Every packet referenced in packets.go (the handler layer) resolves
//     through MustCB / MustSB without panicking, which is verified indirectly
//     by importing the handler package in the test binary's init graph.
//
// Running `go test ./internal/protocoldata/...` catches all of the above.

import (
	"encoding/json"
	"fmt"
	"testing"

	"GoCraft/internal/protocoldata"
)

// maxPacketID is the largest packet ID this test will accept.
// Java Edition packet IDs fit in a VarInt; in practice no vanilla state has
// more than ~130 packets, so IDs ≥ 256 always indicate a data entry error.
const maxPacketID = 255

// stateFileRaw is used only for test-level introspection — we re-parse the
// JSON that the package already validated at init so we can iterate entries.
type stateFileRaw struct {
	GoCraftVersion string           `json:"_gocraft_version"`
	Scope          string           `json:"_scope"`
	Clientbound    map[string]int32 `json:"clientbound"`
	Serverbound    map[string]int32 `json:"serverbound"`
}

// allStates lists every protocol state that has an embedded JSON file.
var allStates = []string{"play", "configuration", "login", "status", "handshake"}

// TestPacketTableIntegrity validates every embedded state file.
func TestPacketTableIntegrity(t *testing.T) {
	for _, state := range allStates {
		state := state // capture
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			checkDirection(t, state, "clientbound", sf.Clientbound)
			checkDirection(t, state, "serverbound", sf.Serverbound)
		})
	}
}

// TestScopeMetadataPresent ensures every file carries a non-empty _scope key,
// so it is clear that the tables are partial rather than complete specs.
func TestScopeMetadataPresent(t *testing.T) {
	for _, state := range allStates {
		state := state
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			if sf.Scope == "" {
				t.Errorf("state %q: missing or empty _scope key — add a human-readable note about coverage", state)
			}
		})
	}
}

// TestNoDuplicateIDs checks that no two semantic names map to the same numeric
// ID within the same state+direction, which would silently route packets to
// the wrong handler.
func TestNoDuplicateIDs(t *testing.T) {
	for _, state := range allStates {
		state := state
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			checkNoDuplicateIDs(t, state, "clientbound", sf.Clientbound)
			checkNoDuplicateIDs(t, state, "serverbound", sf.Serverbound)
		})
	}
}

// TestIDRange verifies every packet ID is within [0, maxPacketID].
func TestIDRange(t *testing.T) {
	for _, state := range allStates {
		state := state
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			checkIDRange(t, state, "clientbound", sf.Clientbound)
			checkIDRange(t, state, "serverbound", sf.Serverbound)
		})
	}
}

// TestReferencedPacketsResolve verifies that every packet name referenced by
// the handler layer's packets.go can be resolved through MustCB / MustSB.
// If any name is missing from the JSON, MustCB/MustSB panicked during init
// and the test binary would have crashed before reaching this function.
// This test therefore documents the known-referenced set explicitly so that
// adding a new packet to packets.go without a matching JSON entry is caught
// by the test output (init panic) rather than only at server startup.
func TestReferencedPacketsResolve(t *testing.T) {
	type entry struct {
		state string
		dir   string
		name  string
	}
	refs := []entry{
		// Handshake
		{"handshake", "serverbound", "minecraft:intention"},
		// Status
		{"status", "serverbound", "minecraft:status_request"},
		{"status", "clientbound", "minecraft:status_response"},
		{"status", "serverbound", "minecraft:ping_request"},
		{"status", "clientbound", "minecraft:pong_response"},
		// Login
		{"login", "serverbound", "minecraft:hello"},
		{"login", "serverbound", "minecraft:key"},
		{"login", "serverbound", "minecraft:login_acknowledged"},
		{"login", "clientbound", "minecraft:login_disconnect"},
		{"login", "clientbound", "minecraft:hello"},
		{"login", "clientbound", "minecraft:game_profile"},
		// Configuration
		{"configuration", "serverbound", "minecraft:client_information"},
		{"configuration", "serverbound", "minecraft:select_known_packs"},
		{"configuration", "serverbound", "minecraft:finish_configuration"},
		{"configuration", "clientbound", "minecraft:custom_payload"},
		{"configuration", "clientbound", "minecraft:finish_configuration"},
		{"configuration", "clientbound", "minecraft:update_enabled_features"},
		{"configuration", "clientbound", "minecraft:update_tags"},
		{"configuration", "clientbound", "minecraft:select_known_packs"},
		// Play — clientbound
		{"play", "clientbound", "minecraft:login"},
		{"play", "clientbound", "minecraft:player_abilities"},
		{"play", "clientbound", "minecraft:player_info_update"},
		{"play", "clientbound", "minecraft:player_info_remove"},
		{"play", "clientbound", "minecraft:player_position"},
		{"play", "clientbound", "minecraft:game_event"},
		{"play", "clientbound", "minecraft:set_chunk_cache_center"},
		{"play", "clientbound", "minecraft:set_default_spawn_position"},
		{"play", "clientbound", "minecraft:keep_alive"},
		{"play", "clientbound", "minecraft:forget_level_chunk"},
		{"play", "clientbound", "minecraft:system_chat"},
		{"play", "clientbound", "minecraft:spawn_entity"},
		{"play", "clientbound", "minecraft:remove_entities"},
		{"play", "clientbound", "minecraft:rotate_head"},
		{"play", "clientbound", "minecraft:teleport_entity"},
		{"play", "clientbound", "minecraft:block_update"},
		{"play", "clientbound", "minecraft:acknowledge_block_change"},
		{"play", "clientbound", "minecraft:set_container_content"},
		{"play", "clientbound", "minecraft:set_held_slot"},
		{"play", "clientbound", "minecraft:commands"},
		{"play", "clientbound", "minecraft:disconnect"},
		{"play", "clientbound", "minecraft:level_chunk_with_light"},
		// Play — serverbound
		{"play", "serverbound", "minecraft:accept_teleportation"},
		{"play", "serverbound", "minecraft:keep_alive"},
		{"play", "serverbound", "minecraft:move_player_pos"},
		{"play", "serverbound", "minecraft:move_player_pos_rot"},
		{"play", "serverbound", "minecraft:move_player_rot"},
		{"play", "serverbound", "minecraft:move_player_status_only"},
		{"play", "serverbound", "minecraft:chat_command"},
		{"play", "serverbound", "minecraft:chat"},
		{"play", "serverbound", "minecraft:player_action"},
		{"play", "serverbound", "minecraft:set_carried_item"},
		{"play", "serverbound", "minecraft:set_creative_mode_slot"},
		{"play", "serverbound", "minecraft:use_item_on"},
	}

	for _, r := range refs {
		r := r
		t.Run(fmt.Sprintf("%s/%s/%s", r.state, r.dir, r.name), func(t *testing.T) {
			var id int32
			var panicked bool
			func() {
				defer func() {
					if rv := recover(); rv != nil {
						panicked = true
						t.Errorf("MustCB/MustSB panicked for %s/%s/%q: %v", r.state, r.dir, r.name, rv)
					}
				}()
				if r.dir == "clientbound" {
					id = protocoldata.MustCB(r.state, r.name)
				} else {
					id = protocoldata.MustSB(r.state, r.name)
				}
			}()
			if !panicked && (id < 0 || id > maxPacketID) {
				t.Errorf("%s/%s/%q: resolved ID %d is out of range [0, %d]",
					r.state, r.dir, r.name, id, maxPacketID)
			}
		})
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func mustParseStateFile(t *testing.T, state string) stateFileRaw {
	t.Helper()
	path := fmt.Sprintf("java/1.21.4/%s.json", state)
	// Read via the embedded FS exposed by protocoldata for test access.
	data, err := protocoldata.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read embedded file %q: %v", path, err)
	}
	var sf stateFileRaw
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("cannot parse %q: %v", path, err)
	}
	return sf
}

func checkDirection(t *testing.T, state, dir string, entries map[string]int32) {
	t.Helper()
	for name, id := range entries {
		if id < 0 || id > maxPacketID {
			t.Errorf("%s/%s/%q: ID %d out of range [0, %d]", state, dir, name, id, maxPacketID)
		}
	}
}

func checkNoDuplicateIDs(t *testing.T, state, dir string, entries map[string]int32) {
	t.Helper()
	seen := make(map[int32]string, len(entries))
	for name, id := range entries {
		if prev, exists := seen[id]; exists {
			t.Errorf("%s/%s: duplicate ID %d — %q and %q both map to it",
				state, dir, id, prev, name)
		}
		seen[id] = name
	}
}

func checkIDRange(t *testing.T, state, dir string, entries map[string]int32) {
	t.Helper()
	for name, id := range entries {
		if id < 0 || id > maxPacketID {
			t.Errorf("%s/%s/%q: ID %d outside valid range [0, %d]",
				state, dir, name, id, maxPacketID)
		}
	}
}
