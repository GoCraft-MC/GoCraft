package handler

// Block interaction handling for Milestone 8.
//
// Receives C→S Player Action (digging) and Use Item On (block placement)
// packets, mutates the canonical core/world, and broadcasts Block Update to
// every connected player.
//
// Placement is acknowledged (so the client does not desync) but not applied
// until M10 supplies inventory and item-in-hand tracking.
//
// All packet IDs are estimates for 1.21.4 (protocol 769); verify against a
// packet capture if block interaction behaves incorrectly.

import (
	"fmt"
	"log/slog"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

// digBreaksBlock reports whether the given Player Action status should break
// the targeted block for a player in the given game mode.
//
// Creative mode breaks on START_DIGGING (status 0) because the client does
// not run a mining animation — the block disappears immediately.
// Survival mode (and adventure) breaks on FINISH_DIGGING (status 2) after the
// full mining animation completes.
//
// In both cases CANCEL_DIGGING (status 1) is left to the caller; no world
// change is made.
func digBreaksBlock(status int32, mode player.GameMode) bool {
	switch mode {
	case player.GameModeCreative:
		return status == actionStatusStartDigging
	default: // survival, adventure
		return status == actionStatusFinishDigging
	}
}

// ── Block packet IDs ──────────────────────────────────────────────────────────

const (
	// Serverbound (C→S)

	// packetIDPlayerAction (C→S) — reports digging events (start, cancel, finish).
	// Estimate for 1.21.4; verify against packet dump.
	packetIDPlayerAction = 0x23

	// packetIDUseItemOn (C→S) — reports block-placement interactions.
	// Estimate for 1.21.4; verify against packet dump.
	packetIDUseItemOn = 0x36

	// Clientbound (S→C)

	// packetIDAcknowledgeBlockChange (S→C) — echoes the client's sequence ID
	// to confirm that a block change was accepted by the server.
	// Confident; present and stable since 1.19.
	packetIDAcknowledgeBlockChange = 0x05

	// packetIDBlockUpdate (S→C) — updates a single block in the world.
	// Confident; present and stable since very early protocol versions.
	packetIDBlockUpdate = 0x09
)

// Player Action status codes (field "status" in C→S Player Action).
const (
	actionStatusStartDigging  = 0 // block targeted — instant break in creative
	actionStatusCancelDigging = 1 // player looked away / right-clicked before break
	actionStatusFinishDigging = 2 // break animation completed (survival)
)

// ── Dispatch ──────────────────────────────────────────────────────────────────

// handleBlockPacket dispatches an incoming block-interaction packet.
// Called from the play loop for packets that need the world and session manager.
func handleBlockPacket(pkt *protocol.Packet, p *player.Player, w *coreworld.World, mgr *session.Manager) error {
	switch pkt.ID {
	case packetIDPlayerAction:
		return handlePlayerAction(pkt, p, w, mgr)
	case packetIDUseItemOn:
		return handleUseItemOn(pkt, p, mgr)
	}
	return nil
}

// ── C→S handlers ─────────────────────────────────────────────────────────────

// handlePlayerAction handles C→S Player Action.
//
// Wire layout (1.21.4, estimate):
//
//	VarInt    status   (0=start, 1=cancel, 2=finish digging)
//	Long      location (packed block position: X«38 | Z«12 | Y)
//	Byte      face     (0=−Y, 1=+Y, 2=−Z, 3=+Z, 4=−X, 5=+X)
//	VarInt    sequence (monotonic counter; echoed in Acknowledge Block Change)
func handlePlayerAction(pkt *protocol.Packet, p *player.Player, w *coreworld.World, mgr *session.Manager) error {
	r := pkt.Reader()

	status, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("player action: reading status: %w", err)
	}
	bx, by, bz, err := protocol.ReadPosition(r)
	if err != nil {
		return fmt.Errorf("player action: reading position: %w", err)
	}
	if _, err := protocol.ReadByte(r); err != nil { // face — unused in M8
		return fmt.Errorf("player action: reading face: %w", err)
	}
	seq, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("player action: reading sequence: %w", err)
	}

	// Reject out-of-bounds Y before touching the world.
	if int(by) < coreworld.WorldMinY || int(by) > coreworld.WorldMaxY {
		slog.Warn("player action: Y out of bounds", "player", p.Username, "y", by)
		sendAcknowledgeBlockChange(mgr, p, seq)
		return nil
	}

	if digBreaksBlock(status, p.GameMode) {
		slog.Info("block break", "player", p.Username,
			"x", bx, "y", by, "z", bz,
			"mode", p.GameMode, "status", status)
		applyBlockChange(int(bx), int(by), int(bz), coreworld.Air, w, mgr)
	}

	// Always acknowledge so the client does not roll back its optimistic update.
	sendAcknowledgeBlockChange(mgr, p, seq)
	return nil
}

// handleUseItemOn handles C→S Use Item On (block placement).
//
// In M8 we have no inventory tracking, so the packet is parsed for its
// sequence ID and acknowledged without placing a block.  Without the
// acknowledgement the client reverts its optimistic placement, which is
// the correct behaviour for now.
//
// Wire layout (1.21.4, estimate):
//
//	VarInt    hand         (0=main hand, 1=off hand)
//	Long      location     (packed block position)
//	VarInt    face         (0=−Y, 1=+Y, 2=−Z, 3=+Z, 4=−X, 5=+X)
//	Float     cursor_x/y/z (hit position within the target face, 0.0–1.0)
//	Bool      inside_block (player head is inside a block)
//	VarInt    sequence
func handleUseItemOn(pkt *protocol.Packet, p *player.Player, mgr *session.Manager) error {
	r := pkt.Reader()

	if _, err := protocol.ReadVarInt(r); err != nil { // hand
		return fmt.Errorf("use item on: reading hand: %w", err)
	}
	if _, _, _, err := protocol.ReadPosition(r); err != nil { // location
		return fmt.Errorf("use item on: reading position: %w", err)
	}
	if _, err := protocol.ReadVarInt(r); err != nil { // face
		return fmt.Errorf("use item on: reading face: %w", err)
	}
	if _, err := protocol.ReadFloat(r); err != nil { // cursor_x
		return fmt.Errorf("use item on: reading cursor_x: %w", err)
	}
	if _, err := protocol.ReadFloat(r); err != nil { // cursor_y
		return fmt.Errorf("use item on: reading cursor_y: %w", err)
	}
	if _, err := protocol.ReadFloat(r); err != nil { // cursor_z
		return fmt.Errorf("use item on: reading cursor_z: %w", err)
	}
	if _, err := protocol.ReadBool(r); err != nil { // inside_block
		return fmt.Errorf("use item on: reading inside_block: %w", err)
	}
	seq, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("use item on: reading sequence: %w", err)
	}

	// Acknowledge without placing — client correctly reverts its optimistic
	// preview since we do not send a Block Update.
	sendAcknowledgeBlockChange(mgr, p, seq)
	return nil
}

// ── World mutation + broadcast ────────────────────────────────────────────────

// applyBlockChange sets the block at (x, y, z) in the canonical world and
// broadcasts a Block Update packet to every connected player.
func applyBlockChange(x, y, z int, block coreworld.Block, w *coreworld.World, mgr *session.Manager) {
	w.SetBlock(x, y, z, block)
	stateID := javaworld.StateID(block)
	pkt := buildBlockUpdate(x, y, z, stateID)
	for _, s := range mgr.SnapshotAll() {
		_ = s.Conn.WritePacket(pkt)
	}
}

// sendAcknowledgeBlockChange sends an Acknowledge Block Change packet to the
// session identified by p.UUID.
func sendAcknowledgeBlockChange(mgr *session.Manager, p *player.Player, seq int32) {
	sess, ok := mgr.Get(p.UUID)
	if !ok {
		return
	}
	_ = sess.Conn.WritePacket(buildAcknowledgeBlockChange(seq))
}

// ── Packet builders ───────────────────────────────────────────────────────────

// buildBlockUpdate constructs a Block Update (S→C) packet.
//
// Wire layout (1.21.4):
//
//	Long    location (packed: X«38 | Z«12 | Y)
//	VarInt  block_state_id
func buildBlockUpdate(x, y, z int, stateID int32) *protocol.Packet {
	return protocol.NewBuilder(packetIDBlockUpdate).
		Long(packBlockPos(x, y, z)).
		VarInt(stateID).
		Build()
}

// buildAcknowledgeBlockChange constructs an Acknowledge Block Change (S→C) packet.
//
// Wire layout (1.21.4):
//
//	VarInt  sequence_id
func buildAcknowledgeBlockChange(seq int32) *protocol.Packet {
	return protocol.NewBuilder(packetIDAcknowledgeBlockChange).
		VarInt(seq).
		Build()
}

// packBlockPos encodes absolute block coordinates as the Minecraft 64-bit
// packed Position: X(26 bits) | Z(26 bits) | Y(12 bits).
func packBlockPos(x, y, z int) int64 {
	return ((int64(x) & 0x3FFFFFF) << 38) |
		((int64(z) & 0x3FFFFFF) << 12) |
		(int64(y) & 0xFFF)
}
