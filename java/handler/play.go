package handler

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
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
	packetIDForgetLevelChunk = 0x22 // Forget Level Chunk (S→C)

	// Serverbound (C→S) — Play state
	packetIDConfirmTeleport              = 0x00 // Confirm Teleport
	packetIDClientKeepAlive             = 0x18 // Keep Alive (C→S)
	packetIDSetPlayerPosition            = 0x1A // Set Player Position
	packetIDSetPlayerPositionAndRotation = 0x1B // Set Player Position and Rotation
	packetIDSetPlayerRotation            = 0x1C // Set Player Rotation
	packetIDSetPlayerOnGround            = 0x1D // Set Player On Ground

	// Game Event reasons
	gameEventStartWaitingForChunks = 13
)

// viewRadius is the number of chunks in each direction to load around the player.
// A radius of 3 gives a 7×7 = 49 chunk view square.
const viewRadius = int32(3)

// keepAliveInterval is how often the server sends a Keep Alive to the client.
const keepAliveInterval = 10 * time.Second

// keepAliveTimeout is how long the server waits for a Keep Alive response.
const keepAliveTimeout = 30 * time.Second

// HandlePlay sends the initial Play-state packet burst, waits for the client
// to confirm the teleport, sends nearby chunks, then runs the keep-alive /
// packet loop until the client disconnects or the connection errors.
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
//	S→C  Level Chunk With Light    (0x27) × (2·viewRadius+1)² — initial burst
//	     … keep-alive / movement / play loop …
func HandlePlay(conn *network.ClientConn, p *player.Player, w *coreworld.World, sender *javaworld.Sender, mgr *session.Manager, cmds *Dispatcher) error {
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
	// Send initial inventory state and confirm the active hotbar slot.
	if err := sendSetContainerContent(conn, p, 1); err != nil {
		return fmt.Errorf("play: %w", err)
	}
	if err := sendSetHeldItem(conn, p.HeldSlot); err != nil {
		return fmt.Errorf("play: %w", err)
	}
	// Send command graph for tab completion.
	if err := conn.WritePacket(buildCommandsPacket()); err != nil {
		return fmt.Errorf("play: %w", err)
	}

	slog.Info("player entered play state",
		"remote", conn.RemoteAddr(),
		"name", p.Username,
		"uuid", p.UUID,
	)

	return playLoop(conn, p, teleportID, w, sender, mgr, cmds)
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
		VarInt(int32(p.GameMode)).      // gamemode (0=survival, 1=creative, etc.)
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
	chunkX := posToChunk(p.Position.X)
	chunkZ := posToChunk(p.Position.Z)
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

// sendForgetChunk sends Forget Level Chunk (0x22 S→C), instructing the client
// to unload the given chunk column.
// Wire order for this packet is Z then X.
func sendForgetChunk(conn *network.ClientConn, cx, cz int32) error {
	return conn.WritePacket(protocol.NewBuilder(packetIDForgetLevelChunk).
		Int(cz).Int(cx).Build())
}

// ── Play loop ─────────────────────────────────────────────────────────────────

// playLoop is the main body for an in-game player session.
//
// After the teleport is confirmed it registers the session with mgr, announces
// the join to other players, sends the initial chunk burst, then enters a tight
// loop that:
//   - Sends periodic Keep Alive packets and validates the client's response.
//   - Reads and dispatches incoming packets (movement, keep-alive, etc.).
//   - Broadcasts position updates to all other sessions on every movement packet.
//   - Streams new chunks and unloads old chunks whenever the player crosses a
//     chunk boundary.
//
// On exit the session is removed from mgr and all other players are notified.
func playLoop(conn *network.ClientConn, p *player.Player, spawnTeleportID int32, w *coreworld.World, sender *javaworld.Sender, mgr *session.Manager, cmds *Dispatcher) error {
	// Must receive Confirm Teleport for the spawn position before anything else.
	if err := readConfirmTeleport(conn, spawnTeleportID); err != nil {
		return fmt.Errorf("play loop: %w", err)
	}

	// ── Multiplayer registration ─────────────────────────────────────────────
	// Register before announcing so ForEachExcept in onPlayerJoin can see us,
	// and the joiner can see existing players.
	// Defers run LIFO: onPlayerLeave fires first (broadcasts while still in map),
	// then Remove cleans up the map entry.
	sess := &session.Session{Player: p, Conn: conn}
	mgr.Add(sess)
	defer mgr.Remove(p.UUID)
	defer onPlayerLeave(mgr, sess)
	onPlayerJoin(mgr, sess, w.Entities.Snapshot())

	// ── Initial chunk burst ──────────────────────────────────────────────────
	chunkX := posToChunk(p.Position.X)
	chunkZ := posToChunk(p.Position.Z)

	// sentChunks tracks which chunk columns the client currently has loaded.
	// Keyed by [cx, cz] for O(1) membership tests on boundary crossings.
	sentChunks := make(map[[2]int32]struct{})

	for dx := -viewRadius; dx <= viewRadius; dx++ {
		for dz := -viewRadius; dz <= viewRadius; dz++ {
			cx, cz := chunkX+dx, chunkZ+dz
			c := w.Chunk(cx, cz)
			if err := sender.SendChunk(conn, c); err != nil {
				return fmt.Errorf("play loop: initial chunk (%d,%d): %w", cx, cz, err)
			}
			sentChunks[[2]int32{cx, cz}] = struct{}{}
		}
	}

	lastChunkX, lastChunkZ := chunkX, chunkZ

	// ── Keep-alive state ─────────────────────────────────────────────────────
	var (
		keepAliveSeq   atomic.Int64
		lastKASent     time.Time
		pendingAliveID int64 = -1 // -1 = no outstanding keep-alive
	)
	lastKASent = time.Now()

	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	// ── Main loop ────────────────────────────────────────────────────────────
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

		posChanged, err := handlePlayPacket(pkt, p, &pendingAliveID)
		if err != nil {
			return fmt.Errorf("play loop: packet 0x%02X: %w", pkt.ID, err)
		}

		// Broadcast position/rotation to all other sessions on every movement.
		if posChanged {
			broadcastPosition(mgr, p)
		}

		// Chat and commands need the session manager and dispatcher.
		if pkt.ID == packetIDChatMessage || pkt.ID == packetIDChatCommand {
			if err := handleChatPacket(pkt, p, mgr, cmds, w, conn); err != nil {
				slog.Warn("chat error", "player", p.Username, "err", err)
			}
		}

		// Block interaction needs both the world and the session manager.
		if pkt.ID == packetIDPlayerAction || pkt.ID == packetIDUseItemOn {
			if err := handleBlockPacket(pkt, p, w, mgr); err != nil {
				slog.Warn("block interaction error", "player", p.Username, "err", err)
			}
		}

		// Inventory management (held item, creative set item).
		if pkt.ID == packetIDSetHeldItemCS || pkt.ID == packetIDCreativeModeSetItem {
			if err := handleInventoryPacket(pkt, p); err != nil {
				slog.Warn("inventory error", "player", p.Username, "err", err)
			}
		}

		// ── Chunk streaming on boundary crossing ─────────────────────────────
		newChunkX := posToChunk(p.Position.X)
		newChunkZ := posToChunk(p.Position.Z)
		if newChunkX != lastChunkX || newChunkZ != lastChunkZ {
			if err := sendSetCenterChunk(conn, p); err != nil {
				return fmt.Errorf("play loop: center chunk: %w", err)
			}
			if err := updateChunkView(conn, w, sender, sentChunks, lastChunkX, lastChunkZ, newChunkX, newChunkZ); err != nil {
				return fmt.Errorf("play loop: streaming chunks: %w", err)
			}
			lastChunkX, lastChunkZ = newChunkX, newChunkZ
		}
	}
}

// updateChunkView sends chunks newly entering the view square and unloads
// chunks leaving it when the player moves from (oldCX,oldCZ) to (newCX,newCZ).
//
// Only chunks that are in the new square but not in the sent set are sent.
// Only chunks that are in the old square but no longer in the new square are
// forgotten. This keeps network traffic proportional to movement speed rather
// than reloading the entire view on every step.
func updateChunkView(
	conn *network.ClientConn,
	w *coreworld.World,
	sender *javaworld.Sender,
	sent map[[2]int32]struct{},
	oldCX, oldCZ, newCX, newCZ int32,
) error {
	// Send chunks that entered the view.
	for dx := -viewRadius; dx <= viewRadius; dx++ {
		for dz := -viewRadius; dz <= viewRadius; dz++ {
			key := [2]int32{newCX + dx, newCZ + dz}
			if _, ok := sent[key]; ok {
				continue // already loaded on the client
			}
			c := w.Chunk(key[0], key[1])
			if err := sender.SendChunk(conn, c); err != nil {
				return fmt.Errorf("chunk (%d,%d): %w", key[0], key[1], err)
			}
			sent[key] = struct{}{}
		}
	}

	// Unload chunks that left the view.
	for dx := -viewRadius; dx <= viewRadius; dx++ {
		for dz := -viewRadius; dz <= viewRadius; dz++ {
			key := [2]int32{oldCX + dx, oldCZ + dz}
			if _, ok := sent[key]; !ok {
				continue
			}
			// Still within the new view square?
			if abs32(key[0]-newCX) <= viewRadius && abs32(key[1]-newCZ) <= viewRadius {
				continue
			}
			if err := sendForgetChunk(conn, key[0], key[1]); err != nil {
				return fmt.Errorf("forgetting chunk (%d,%d): %w", key[0], key[1], err)
			}
			delete(sent, key)
		}
	}
	return nil
}

// handlePlayPacket dispatches a single incoming Play-state packet.
// Returns (true, nil) when the player's position or rotation was updated so the
// caller can broadcast the change to other sessions.
// Returns a non-nil error only for fatal protocol violations.
func handlePlayPacket(pkt *protocol.Packet, p *player.Player, pendingAliveID *int64) (posChanged bool, err error) {
	switch pkt.ID {
	case packetIDClientKeepAlive:
		// Client echoes the Long ID we sent; validate it.
		if len(pkt.Data) < 8 {
			return false, fmt.Errorf("keep-alive packet too short: %d bytes", len(pkt.Data))
		}
		id := int64(binary.BigEndian.Uint64(pkt.Data[:8]))
		if id == *pendingAliveID {
			*pendingAliveID = -1
		}

	case packetIDSetPlayerPosition:
		// C→S: x, y, z (Double×3), flags (Byte; bit 0 = on_ground)
		r := pkt.Reader()
		x, err := protocol.ReadDouble(r)
		if err != nil {
			return false, fmt.Errorf("reading position x: %w", err)
		}
		y, err := protocol.ReadDouble(r)
		if err != nil {
			return false, fmt.Errorf("reading position y: %w", err)
		}
		z, err := protocol.ReadDouble(r)
		if err != nil {
			return false, fmt.Errorf("reading position z: %w", err)
		}
		flags, err := protocol.ReadByte(r)
		if err != nil {
			return false, fmt.Errorf("reading position flags: %w", err)
		}
		p.Position.X, p.Position.Y, p.Position.Z = x, y, z
		p.OnGround = flags&0x01 != 0
		return true, nil

	case packetIDSetPlayerPositionAndRotation:
		// C→S: x, y, z (Double×3), yaw, pitch (Float×2), flags (Byte)
		r := pkt.Reader()
		x, err := protocol.ReadDouble(r)
		if err != nil {
			return false, fmt.Errorf("reading position x: %w", err)
		}
		y, err := protocol.ReadDouble(r)
		if err != nil {
			return false, fmt.Errorf("reading position y: %w", err)
		}
		z, err := protocol.ReadDouble(r)
		if err != nil {
			return false, fmt.Errorf("reading position z: %w", err)
		}
		yaw, err := protocol.ReadFloat(r)
		if err != nil {
			return false, fmt.Errorf("reading yaw: %w", err)
		}
		pitch, err := protocol.ReadFloat(r)
		if err != nil {
			return false, fmt.Errorf("reading pitch: %w", err)
		}
		flags, err := protocol.ReadByte(r)
		if err != nil {
			return false, fmt.Errorf("reading movement flags: %w", err)
		}
		p.Position.X, p.Position.Y, p.Position.Z = x, y, z
		p.Rotation.Yaw, p.Rotation.Pitch = yaw, pitch
		p.OnGround = flags&0x01 != 0
		return true, nil

	case packetIDSetPlayerRotation:
		// C→S: yaw, pitch (Float×2), flags (Byte)
		r := pkt.Reader()
		yaw, err := protocol.ReadFloat(r)
		if err != nil {
			return false, fmt.Errorf("reading yaw: %w", err)
		}
		pitch, err := protocol.ReadFloat(r)
		if err != nil {
			return false, fmt.Errorf("reading pitch: %w", err)
		}
		flags, err := protocol.ReadByte(r)
		if err != nil {
			return false, fmt.Errorf("reading rotation flags: %w", err)
		}
		p.Rotation.Yaw, p.Rotation.Pitch = yaw, pitch
		p.OnGround = flags&0x01 != 0
		return true, nil

	case packetIDSetPlayerOnGround:
		// C→S: flags (Byte; bit 0 = on_ground)
		if len(pkt.Data) >= 1 {
			p.OnGround = pkt.Data[0]&0x01 != 0
		}

	case packetIDConfirmTeleport:
		// Late teleport confirm — ignore, already processed.

	default:
		// Silently drop all other play packets (chat, interaction, etc.).
		// Future milestones will register handlers here.
	}
	return false, nil
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

// ── Helpers ──────────────────────────────────────────────────────────────────

// posToChunk converts a world coordinate to a chunk coordinate using floor
// division so that negative positions map correctly (e.g. X=-1 → chunk -1).
func posToChunk(pos float64) int32 {
	return int32(math.Floor(pos)) >> 4
}

// abs32 returns the absolute value of n.
func abs32(n int32) int32 {
	if n < 0 {
		return -n
	}
	return n
}
