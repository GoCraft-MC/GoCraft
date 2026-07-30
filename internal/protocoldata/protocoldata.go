// Package protocoldata loads Minecraft Java Edition protocol packet IDs from
// versioned JSON files embedded at compile time and exposes them through a
// startup-validated lookup API.
//
// Usage:
//
//	import "GoCraft/internal/protocoldata"
//
//	var pktLogin = protocoldata.MustCB("play", "minecraft:login")
//	var pktChat  = protocoldata.MustSB("play", "minecraft:chat")
//
// Both functions panic at program startup if the requested packet name is not
// present in the loaded data, ensuring that every handler that references a
// packet is verified against the actual protocol file during development.
//
// Updating IDs for a new protocol version only requires replacing the JSON
// files under java/<version>/ and rebuilding — no Go source changes.
package protocoldata

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
)

// expectedVersion is the Minecraft version whose protocol data is embedded.
const expectedVersion = "1.21.4"

// ── Embedded files ────────────────────────────────────────────────────────────

//go:embed java/1.21.4/play.json java/1.21.4/configuration.json java/1.21.4/login.json java/1.21.4/status.json java/1.21.4/handshake.json
var fs embed.FS

// ── In-memory tables ──────────────────────────────────────────────────────────

// packetTable maps "state\x00direction\x00name" → numeric packet ID.
// Built once at init from the embedded JSON files.
var packetTable map[string]int32

// ── JSON schema ───────────────────────────────────────────────────────────────

type stateFile struct {
	GoCraftVersion string           `json:"_gocraft_version"`
	Clientbound    map[string]int32 `json:"clientbound"`
	Serverbound    map[string]int32 `json:"serverbound"`
}

// ── Init ──────────────────────────────────────────────────────────────────────

func init() {
	states := []string{"play", "configuration", "login", "status", "handshake"}
	packetTable = make(map[string]int32, 128)

	for _, state := range states {
		path := fmt.Sprintf("java/%s/%s.json", expectedVersion, state)
		data, err := fs.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("protocoldata: reading %s: %v", path, err))
		}

		var sf stateFile
		if err := json.Unmarshal(data, &sf); err != nil {
			panic(fmt.Sprintf("protocoldata: parsing %s: %v", path, err))
		}
		if sf.GoCraftVersion != expectedVersion {
			panic(fmt.Sprintf(
				"protocoldata: %s declares version %q but server expects %q",
				path, sf.GoCraftVersion, expectedVersion))
		}

		for name, id := range sf.Clientbound {
			if name == "_comment" || name == "_source" {
				continue
			}
			packetTable[tableKey(state, "clientbound", name)] = id
		}
		for name, id := range sf.Serverbound {
			if name == "_comment" || name == "_source" {
				continue
			}
			packetTable[tableKey(state, "serverbound", name)] = id
		}
	}

	total := len(packetTable)
	slog.Info("protocoldata: loaded protocol packet IDs",
		"version", expectedVersion, "packets", total)
}

// ── Public API ────────────────────────────────────────────────────────────────

// MustCB returns the clientbound (S→C) numeric packet ID for the given
// protocol state and canonical packet name.
//
// Panics at startup if the name is not found in the embedded protocol data.
// This is intentional: a missing packet name is a programming error that must
// be caught before the server accepts connections.
func MustCB(state, name string) int32 {
	return mustLookup(state, "clientbound", name)
}

// MustSB returns the serverbound (C→S) numeric packet ID for the given
// protocol state and canonical packet name.
//
// Panics at startup if the name is not found in the embedded protocol data.
func MustSB(state, name string) int32 {
	return mustLookup(state, "serverbound", name)
}

// mustLookup panics with a clear message if name is not in the table.
func mustLookup(state, dir, name string) int32 {
	key := tableKey(state, dir, name)
	id, ok := packetTable[key]
	if !ok {
		panic(fmt.Sprintf(
			"protocoldata: unknown packet %s/%s/%q — add it to java/%s/%s.json",
			state, dir, name, expectedVersion, state))
	}
	return id
}

// tableKey encodes (state, direction, name) as a single map key.
func tableKey(state, dir, name string) string {
	return state + "\x00" + dir + "\x00" + name
}
