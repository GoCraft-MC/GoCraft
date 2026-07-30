package handler

// Commands packet builder for Milestone 12.
//
// The Commands packet (S→C 0x11) sends the server's command graph to the
// client as a brigadier directed acyclic graph.  The client uses it to drive
// tab completion and the / command input screen.
//
// Each node carries:
//   - A flags byte  (type | executable | redirect | suggestions)
//   - A VarInt array of child node indices
//   - For LITERAL nodes: a String name
//   - For ARGUMENT nodes: a String name + VarInt parser ID + parser properties
//
// Node types:
//   0x00  ROOT     — single root; no name; children = top-level commands
//   0x01  LITERAL  — a fixed keyword (e.g. "gamemode")
//   0x02  ARGUMENT — a typed parameter (e.g. brigadier:integer)
//
// Flags layout:
//   bits 0-1  node type
//   bit  2    is_executable  (client shows ↵ hint when all required args satisfied)
//   bit  3    has_redirect   (not used here)
//   bit  4    has_suggestions (not used here)
//
// Packet ID 0x11 is an estimate for 1.21.4 (protocol 769).
// Milestone 13 will replace hardcoded IDs with data-driven registry loading.

import "GoCraft/java/protocol"

const packetIDCommands = 0x11 // S→C Play, estimate for 1.21.4

// buildCommandsPacket constructs the Commands S→C packet for the GoCraft
// built-in command set:
//
//	/gamemode <survival|creative|adventure|spectator>
//	/tp <x> <y> <z>
//	/tp <player>
//	/give <item> [count]
//	/kick <player> [reason]
//	/help
//	/list
func buildCommandsPacket() *protocol.Packet {
	// ── Node index constants ──────────────────────────────────────────────────
	const (
		nRoot       int32 = 0
		nGamemode   int32 = 1
		nSurvival   int32 = 2
		nCreative   int32 = 3
		nAdventure  int32 = 4
		nSpectator  int32 = 5
		nTp         int32 = 6
		nTpX        int32 = 7
		nTpY        int32 = 8
		nTpZ        int32 = 9
		nTpTarget   int32 = 10
		nGive       int32 = 11
		nGiveItem   int32 = 12
		nGiveCount  int32 = 13
		nKick       int32 = 14
		nKickTarget int32 = 15
		nKickReason int32 = 16
		nHelp       int32 = 17
		nList       int32 = 18
		nodeCount         = 19

		// Brigadier built-in parser IDs (stable across Minecraft versions).
		parserDouble  int32 = 2
		parserInteger int32 = 3
		parserString  int32 = 5

		// brigadier:string behaviour sub-types.
		strSingleWord int32 = 0
		strGreedy     int32 = 2

		// Flags shortcuts.
		fRoot     byte = 0x00                 // ROOT, not executable
		fLit      byte = 0x01                 // LITERAL, not executable
		fLitX     byte = 0x01 | 0x04          // LITERAL, executable
		fArg      byte = 0x02                 // ARGUMENT, not executable
		fArgX     byte = 0x02 | 0x04          // ARGUMENT, executable
	)

	b := protocol.NewBuilder(packetIDCommands)
	b.VarInt(nodeCount)

	// ── 0: ROOT ───────────────────────────────────────────────────────────────
	b.Byte(fRoot)
	nodeChildren(b, nGamemode, nTp, nGive, nKick, nHelp, nList)

	// ── 1: LITERAL "gamemode" ─────────────────────────────────────────────────
	b.Byte(fLit)
	nodeChildren(b, nSurvival, nCreative, nAdventure, nSpectator)
	b.String("gamemode")

	// ── 2–5: game-mode sub-commands ───────────────────────────────────────────
	for _, name := range []string{"survival", "creative", "adventure", "spectator"} {
		b.Byte(fLitX)
		nodeChildren(b) // leaf
		b.String(name)
	}

	// ── 6: LITERAL "tp" ──────────────────────────────────────────────────────
	b.Byte(fLit)
	nodeChildren(b, nTpX, nTpTarget)
	b.String("tp")

	// ── 7: ARGUMENT "x" (double) ─────────────────────────────────────────────
	b.Byte(fArg)
	nodeChildren(b, nTpY)
	b.String("x").VarInt(parserDouble).Byte(0x00) // no min/max bounds

	// ── 8: ARGUMENT "y" (double) ─────────────────────────────────────────────
	b.Byte(fArg)
	nodeChildren(b, nTpZ)
	b.String("y").VarInt(parserDouble).Byte(0x00)

	// ── 9: ARGUMENT "z" (double, executable) ─────────────────────────────────
	b.Byte(fArgX)
	nodeChildren(b)
	b.String("z").VarInt(parserDouble).Byte(0x00)

	// ── 10: ARGUMENT "target" (string single-word, executable) ───────────────
	b.Byte(fArgX)
	nodeChildren(b)
	b.String("target").VarInt(parserString).VarInt(strSingleWord)

	// ── 11: LITERAL "give" ────────────────────────────────────────────────────
	b.Byte(fLit)
	nodeChildren(b, nGiveItem)
	b.String("give")

	// ── 12: ARGUMENT "item" (string single-word, executable so count is
	//        optional)  ────────────────────────────────────────────────────────
	b.Byte(fArgX)
	nodeChildren(b, nGiveCount)
	b.String("item").VarInt(parserString).VarInt(strSingleWord)

	// ── 13: ARGUMENT "count" (integer ≥ 1, executable) ───────────────────────
	b.Byte(fArgX)
	nodeChildren(b)
	b.String("count").VarInt(parserInteger)
	b.Byte(0x01).Int(1) // flags: min present; min = 1

	// ── 14: LITERAL "kick" ────────────────────────────────────────────────────
	b.Byte(fLit)
	nodeChildren(b, nKickTarget)
	b.String("kick")

	// ── 15: ARGUMENT "target" (string single-word, executable so reason is
	//        optional) ─────────────────────────────────────────────────────────
	b.Byte(fArgX)
	nodeChildren(b, nKickReason)
	b.String("target").VarInt(parserString).VarInt(strSingleWord)

	// ── 16: ARGUMENT "reason" (string greedy-phrase, executable) ─────────────
	b.Byte(fArgX)
	nodeChildren(b)
	b.String("reason").VarInt(parserString).VarInt(strGreedy)

	// ── 17: LITERAL "help" (executable) ──────────────────────────────────────
	b.Byte(fLitX)
	nodeChildren(b)
	b.String("help")

	// ── 18: LITERAL "list" (executable) ──────────────────────────────────────
	b.Byte(fLitX)
	nodeChildren(b)
	b.String("list")

	// Root node index (always 0).
	b.VarInt(nRoot)

	return b.Build()
}

// nodeChildren writes a VarInt child-count followed by each child index.
func nodeChildren(b *protocol.Builder, indices ...int32) {
	b.VarInt(int32(len(indices)))
	for _, i := range indices {
		b.VarInt(i)
	}
}
