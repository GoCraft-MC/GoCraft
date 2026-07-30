package protocoldata_test

// Validation tests for the embedded protocol-data packet tables.
//
// Each test targets a distinct invariant:
//
//   TestVersionMetadata        — _gocraft_version present and matches expected
//   TestScopeMetadataPresent   — _scope is present and non-empty
//   TestPacketNameFormat        — all names start with "minecraft:" and contain no whitespace
//   TestBothDirectionsPresent   — every state file has both clientbound and serverbound maps
//   TestNoDuplicateIDs          — no two names share a numeric ID in the same direction
//   TestIDRange                 — all IDs within [0, maxPacketID]
//   TestReferencedPacketsResolve— every packet used by GoCraft resolves through MustCB/MustSB
//
// Running `go test ./internal/protocoldata/...` catches all of the above.

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"GoCraft/internal/protocoldata"
)

// maxPacketID is the largest packet ID this test will accept.
// In Java Edition 1.21.4 no play-state direction exceeds ~130 packets,
// so IDs ≥ 256 always indicate a data-entry error.
const maxPacketID = 255

// expectedVersion must match the _gocraft_version field in every JSON file.
const expectedVersion = "1.21.4"

// stateFileRaw is used for test-level introspection — we re-parse the embedded
// JSON so we can iterate every entry individually.
type stateFileRaw struct {
	GoCraftVersion string           `json:"_gocraft_version"`
	Scope          string           `json:"_scope"`
	Clientbound    map[string]int32 `json:"clientbound"`
	Serverbound    map[string]int32 `json:"serverbound"`
}

// allStates lists every protocol state that has an embedded JSON file.
var allStates = []string{"play", "configuration", "login", "status", "handshake"}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestVersionMetadata verifies that every file declares the expected version
// string, catching accidental use of a wrong-version data file.
func TestVersionMetadata(t *testing.T) {
	for _, state := range allStates {
		state := state
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			if sf.GoCraftVersion == "" {
				t.Errorf("state %q: missing _gocraft_version key", state)
			} else if sf.GoCraftVersion != expectedVersion {
				t.Errorf("state %q: _gocraft_version is %q, want %q",
					state, sf.GoCraftVersion, expectedVersion)
			}
		})
	}
}

// TestScopeMetadataPresent ensures every file carries a non-empty _scope key
// so it is unambiguous that the tables are partial, not complete protocol specs.
func TestScopeMetadataPresent(t *testing.T) {
	for _, state := range allStates {
		state := state
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			if sf.Scope == "" {
				t.Errorf("state %q: missing or empty _scope — add a human-readable coverage note", state)
			}
		})
	}
}

// TestPacketNameFormat checks that every semantic packet name:
//   - starts with "minecraft:" (namespace required)
//   - contains no ASCII whitespace (a common copy-paste mistake)
func TestPacketNameFormat(t *testing.T) {
	const prefix = "minecraft:"
	for _, state := range allStates {
		state := state
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			checkNameFormat(t, state, "clientbound", sf.Clientbound, prefix)
			checkNameFormat(t, state, "serverbound", sf.Serverbound, prefix)
		})
	}
}

// TestBothDirectionsPresent verifies that every state file has both
// "clientbound" and "serverbound" top-level maps, even if one is empty
// (e.g. handshake has no clientbound packets).  A missing key would silently
// leave one direction unresolvable.
func TestBothDirectionsPresent(t *testing.T) {
	for _, state := range allStates {
		state := state
		t.Run(state, func(t *testing.T) {
			sf := mustParseStateFile(t, state)
			if sf.Clientbound == nil {
				t.Errorf("state %q: missing \"clientbound\" map (use {} for empty)", state)
			}
			if sf.Serverbound == nil {
				t.Errorf("state %q: missing \"serverbound\" map (use {} for empty)", state)
			}
		})
	}
}

// TestNoDuplicateIDs checks that no two semantic names map to the same numeric
// ID within the same state+direction, which would silently route packets to the
// wrong handler.
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
// GoCraft's handler layer (packets.go) resolves through MustCB / MustSB.
// A missing JSON entry surfaces as a panic+FAIL here instead of only at server
// startup, making CI the gate rather than a production incident.
func TestReferencedPacketsResolve(t *testing.T) {
	type entry struct{ state, dir, name string }
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
						t.Errorf("panicked for %s/%s/%q: %v", r.state, r.dir, r.name, rv)
					}
				}()
				if r.dir == "clientbound" {
					id = protocoldata.MustCB(r.state, r.name)
				} else {
					id = protocoldata.MustSB(r.state, r.name)
				}
			}()
			if !panicked && (id < 0 || id > maxPacketID) {
				t.Errorf("%s/%s/%q: resolved ID %d outside valid range [0, %d]",
					r.state, r.dir, r.name, id, maxPacketID)
			}
		})
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func mustParseStateFile(t *testing.T, state string) stateFileRaw {
	t.Helper()
	path := fmt.Sprintf("java/1.21.4/%s.json", state)
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

func checkNameFormat(t *testing.T, state, dir string, entries map[string]int32, prefix string) {
	t.Helper()
	for name := range entries {
		if !strings.HasPrefix(name, prefix) {
			t.Errorf("%s/%s: name %q must start with %q", state, dir, name, prefix)
		}
		if strings.ContainsAny(name, " \t\n\r") {
			t.Errorf("%s/%s: name %q contains whitespace", state, dir, name)
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
