package handler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"GoCraft/core/player"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

// ── Play state packet IDs (1.21.4 / protocol 769) ────────────────────────────

const (
	// Clientbound (S→C) — Play state
	packetIDPlayLogin        = 0x2C // Login (Play)
	packetIDPlayerAbilities  = 0x3A // Player Abilities
	packetIDPlayerInfoUpdate = 0x40 // Player Info Update (tab list)
	packetIDSyncPosition     = 0x42 // Synchronize Player Position
	packetIDGameEvent        = 0x23 // Game Event
	packetIDSetCenterChunk   = 0x58 // Set Center Chunk (update_view_position)
	packetIDSpawnPosition    = 0x5B // Set Default Spawn Position
	packetIDServerKeepAlive  = 0x26 // Keep Alive (S→C)

	// Serverbound (C→S) — Play state
	packetIDConfirmTeleport = 0x00 // Confirm Teleport
	packetIDClientKeepAlive = 0x18 // Keep Alive (C→S)

	// Game Event reasons
	gameEventStartWaitingForChunks = 13
)

// keepAliveInterval is how often the server sends a Keep Alive to the client.
const keepAliveInterval = 10 * time.Second

// keepAliveTimeout is how long the server waits for a Keep Alive response.
const keepAliveTimeout = 30 * time.Second

// HandlePlay sends the initial Play-state packet burst, waits for the client
// to confirm the teleport, then runs the keep-alive / packet loop until the
// client disconnects or the connection errors.
//
// Protocol flow (1.21.4):
//
//	S→C  Login (Play)              (0x2C) — assigns entity ID, dimension, etc.
//	S→C  Player Abilities          (0x3A) — flight / speed flags
//	S→C  Set Default Spawn Pos     (0x5B) — world spawn marker on map
//	S→C  Player Info Update        (0x40) — add self to tab list
//	S→C  Synchronize Player Pos    (0x42) — spawn co-ordinates + teleport ID 1
//	S→C  Set Center Chunk          (0x58) — chunk streaming anchor
//	S→C  Game Event reason=13      (0x23) — "start waiting for level chunks"
//	C→S  Confirm Teleport (ID 1)   (0x00)
//	     … keep-alive / play loop …
func HandlePlay(conn *network.ClientConn, p *player.Player) error {
	// ── Initial burst ────────────────────────────────────────────────────────
	if err := sendLoginPlay(conn, p); err != nil {
		return fmt.Errorf("play: %w", err)
	}
	if err := sendPlayerAbilities(conn, p); err != nil {
		return fmt.Errorf("play: %w", err)
	}
	if err := sendDefaultSpawnPosition(conn, p); err != nil {
		return fmt.Errorf("play: %w", err)
	}
	if err := sendPlayerInfoUpdate(conn, p); err != nil {
		return fmt.Errorf("play: %w", err)
	}

	const teleportID = int32(1)
	if err := sendSyncPosition(conn, p, teleportID); err != nil {
		return fmt.Errorf("play: %w", err)
	}
	if err := sendSetCenterChunk(conn, p); err != nil {
		return fmt.Errorf("play: %w", err)
	}
	// Tell the client to stop waiting for chunks and show the world.
	if err := sendGameEvent(conn, gameEventStartWaitingForChunks, 0); err != nil {
		return fmt.Errorf("play: %w", err)
	}

	slog.Info("player entered play state",
		"remote", conn.RemoteAddr(),
		"name", p.Username,
		"uuid", p.UUID,
	)
	return playLoop(conn, p, teleportID)
}

// ── Clientbound packet helpers ────────────────────────────────────────────────

// sendLoginPlay sends the Login (Play) packet (0x2C S→C).
//
// Fields (1.21.4 / protocol 769):
//
//	Int     entity_id
//	Bool    is_hardcore
//	VarInt  game_mode        (0 = survival)
//	Byte    prev_game_mode   (0xFF = signed -1 = undefined)
//	VarInt  dimension_count  (1)
//	String  dimension_names[0]  "minecraft:overworld"
//	String  dimension_type      "minecraft:overworld"
//	String  dimension_name      "minecraft:overworld"
//	Long    hashed_seed     (0)
//	VarInt  max_players     (informational)
//	VarInt  view_distance   (10)
//	VarInt  sim_distance    (10)
//	Bool    reduced_debug_info  false
//	Bool    enable_respawn_screen  true
//	Bool    do_limited_crafting   false
//	Bool    is_debug         false
//	Bool    is_flat          false
//	Bool    has_death_location   false
//	VarInt  portal_cooldown  0
//	VarInt  sea_level        63  (overworld sea level)
//	Bool    enforce_secure_chat  false
func sendLoginPlay(conn *network.ClientConn, p *player.Player) error {
	pkt := protocol.NewBuilder(packetIDPlayLogin).
		Int(p.EntityID).
		Bool(false).                    // is_hardcore
		VarInt(0).                      // gamemode: 0 = survival
		Byte(0xFF).                     // prev_gamemode: 0xFF = signed -1 = undefined
		VarInt(1).                      // dimension count
		String("minecraft:overworld").  // dimension names[0]
		String("minecraft:overworld").  // dimension_type (registry key)
		String("minecraft:overworld").  // dimension_name (world name)
		Long(0).                        // hashed_seed
		VarInt(20).                     // max_players (informational)
		VarInt(10).                     // view_distance
		VarInt(10).                     // simulation_distance
		Bool(false).                    // reduced_debug_info
		Bool(true).                     // enable_respawn_screen
		Bool(false).                    // do_limited_crafting
		Bool(false).                    // is_debug (debug world?)
		Bool(false).                    // is_flat (superflat?)
		Bool(false).                    // has_death_location
		VarInt(0).                      // portal_cooldown
		VarInt(63).                     // sea_level (overworld = 63)
		Bool(false).                    // enforce_secure_chat
		Build()
	return conn.WritePacket(pkt)
}

// sendPlayerAbilities sends the Player Abilities packet (0x3A S→C).
//
// Flags bitmask:
//
//	0x01 invulnerable
//	0x02 flying
//	0x04 allow_flying
//	0x08 instant_build (creative)
func sendPlayerAbilities(conn *network.ClientConn, p *player.Player) error {
	var flags byte
	if p.GameMode == player.GameModeCreative {
		flags |= 0x01 | 0x04 | 0x08 // invulnerable + allow_fly + instant_break
	}
	pkt := protocol.NewBuilder(packetIDPlayerAbilities).
		Byte(flags).
		Float(0.05). // flying speed
		Float(0.1).  // fov modifier
		Build()
	return conn.WritePacket(pkt)
}

// sendDefaultSpawnPosition sends the Set Default Spawn Position packet (0x5B S→C).
//
// Fields:
//
//	Long   position (64-bit packed block position)
//	Float  angle    (spawn compass bearing)
func sendDefaultSpawnPosition(conn *network.ClientConn, p *player.Player) error {
	x := int64(p.Position.X)
	y := int64(p.Position.Y)
	z := int64(p.Position.Z)
	// Minecraft 64-bit packed position: X(26 bits) | Z(26 bits) | Y(12 bits)
	packed := ((x & 0x3FFFFFF) << 38) | ((z & 0x3FFFFFF) << 12) | (y & 0xFFF)
	pkt := protocol.NewBuilder(packetIDSpawnPosition).
		Long(packed).
		Float(0).
		Build()
	return conn.WritePacket(pkt)
}

// sendPlayerInfoUpdate sends Player Info Update (0x40 S→C) to add the player
// to the tab list.
//
// Action bitmask (1.21.4):
//
//	0x01  ADD_PLAYER        name + properties
//	0x02  INITIALIZE_CHAT   (omitted — no chat session)
//	0x04  UPDATE_GAME_MODE  game mode
//	0x08  UPDATE_LISTED     show in tab list
//	0x10  UPDATE_LATENCY    ping value
//	0x20  UPDATE_DISPLAY_NAME
func sendPlayerInfoUpdate(conn *network.ClientConn, p *player.Player) error {
	// Actions: ADD_PLAYER (0x01) + UPDATE_LISTED (0x08)
	const actions byte = 0x01 | 0x08

	b := protocol.NewBuilder(packetIDPlayerInfoUpdate).
		Byte(actions).
		VarInt(1).                        // 1 player entry
		UUID(protocol.UUID(p.UUID))       // player UUID

	// ADD_PLAYER (0x01) data: name + 0 properties
	b.String(p.Username).VarInt(0)

	// UPDATE_LISTED (0x08) data: listed = true
	b.Bool(true)

	return conn.WritePacket(b.Build())
}

// sendSyncPosition sends Synchronize Player Position (0x42 S→C).
//
// Fields (1.21.4):
//
//	Double  x, y, z
//	Double  velocity_x, velocity_y, velocity_z
//	Float   yaw, pitch
//	Int     flags (bitmask; 0 = all absolute)
//	VarInt  teleport_id
func sendSyncPosition(conn *network.ClientConn, p *player.Player, teleportID int32) error {
	pkt := protocol.NewBuilder(packetIDSyncPosition).
		Double(p.Position.X).
		Double(p.Position.Y).
		Double(p.Position.Z).
		Double(0). // velocity x
		Double(0). // velocity y
		Double(0). // velocity z
		Float(p.Rotation.Yaw).
		Float(p.Rotation.Pitch).
		Int(0). // flags: 0 = absolute position
		VarInt(teleportID).
		Build()
	return conn.WritePacket(pkt)
}

// sendSetCenterChunk sends Set Center Chunk (0x58 S→C).
// This tells the client which chunk the player is in, controlling which
// chunks are loaded / unloaded around them.
func sendSetCenterChunk(conn *network.ClientConn, p *player.Player) error {
	chunkX := int32(p.Position.X) >> 4
	chunkZ := int32(p.Position.Z) >> 4
	pkt := protocol.NewBuilder(packetIDSetCenterChunk).
		VarInt(chunkX).
		VarInt(chunkZ).
		Build()
	return conn.WritePacket(pkt)
}

// sendGameEvent sends a Game Event packet (0x23 S→C).
//
// Fields:
//
//	Unsigned Byte  reason
//	Float          value
func sendGameEvent(conn *network.ClientConn, reason byte, value float32) error {
	pkt := protocol.NewBuilder(packetIDGameEvent).
		Byte(reason).
		Float(value).
		Build()
	return conn.WritePacket(pkt)
}

// ── Play loop ─────────────────────────────────────────────────────────────────

// playLoop is the main body for an in-game player session.
// It reads incoming packets and dispatches them, while a ticker sends
// periodic Keep Alive packets to the client.
func playLoop(conn *network.ClientConn, p *player.Player, spawnTeleportID int32) error {
	// Must receive Confirm Teleport for the spawn position before anything else.
	if err := readConfirmTeleport(conn, spawnTeleportID); err != nil {
		return fmt.Errorf("play loop: %w", err)
	}

	var (
		keepAliveSeq   atomic.Int64
		lastKASent     time.Time
		pendingAliveID int64 = -1 // -1 = no outstanding keep-alive
	)
	lastKASent = time.Now()

	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		// Check keep-alive timeout.
		if pendingAliveID >= 0 && time.Since(lastKASent) > keepAliveTimeout {
			return fmt.Errorf("play loop: keep-alive timeout for player %s", p.Username)
		}

		// Send keep-alive if the interval has elapsed.
		select {
		case <-ticker.C:
			id := keepAliveSeq.Add(1)
			if err := sendKeepAlive(conn, id); err != nil {
				return fmt.Errorf("play loop: sending keep-alive: %w", err)
			}
			pendingAliveID = id
			lastKASent = time.Now()
		default:
		}

		// Read the next packet (ReadPacket sets its own deadline).
		pkt, err := conn.ReadPacket()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				slog.Info("player disconnected", "name", p.Username, "remote", conn.RemoteAddr())
				return nil
			}
			return fmt.Errorf("play loop: reading packet: %w", err)
		}

		if err := handlePlayPacket(pkt, &pendingAliveID); err != nil {
			return fmt.Errorf("play loop: packet 0x%02X: %w", pkt.ID, err)
		}
	}
}

// handlePlayPacket dispatches a single incoming Play-state packet.
// Returns an error only for fatal protocol violations.
func handlePlayPacket(pkt *protocol.Packet, pendingAliveID *int64) error {
	switch pkt.ID {
	case packetIDClientKeepAlive:
		// Client echoes the Long ID we sent; validate it.
		if len(pkt.Data) < 8 {
			return fmt.Errorf("keep-alive packet too short: %d bytes", len(pkt.Data))
		}
		id := int64(binary.BigEndian.Uint64(pkt.Data[:8]))
		if id == *pendingAliveID {
			*pendingAliveID = -1
		}

	case packetIDConfirmTeleport:
		// Late teleport confirm — ignore, already processed.

	default:
		// Silently drop all other play packets (movement, chat, etc.).
		// Future milestones will register handlers here.
	}
	return nil
}

// readConfirmTeleport reads one Confirm Teleport packet and verifies the ID.
func readConfirmTeleport(conn *network.ClientConn, wantID int32) error {
	pkt, err := conn.ReadPacket()
	if err != nil {
		return fmt.Errorf("reading Confirm Teleport: %w", err)
	}
	if pkt.ID != packetIDConfirmTeleport {
		return fmt.Errorf("expected 0x00 (Confirm Teleport), got 0x%02X", pkt.ID)
	}
	r := pkt.Reader()
	gotID, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("reading teleport ID: %w", err)
	}
	if gotID != wantID {
		return fmt.Errorf("teleport ID mismatch: got %d, want %d", gotID, wantID)
	}
	return nil
}

// sendKeepAlive sends a Keep Alive packet (0x26 S→C) with the given ID.
func sendKeepAlive(conn *network.ClientConn, id int64) error {
	return conn.WritePacket(protocol.NewBuilder(packetIDServerKeepAlive).Long(id).Build())
}
