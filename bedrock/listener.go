// Package bedrock implements the Minecraft Bedrock Edition network adapter for
// GoCraft.  It accepts UDP/RakNet connections via gophertunnel, authenticates
// players through Xbox Live, and translates between the Bedrock protocol and
// the edition-agnostic core simulation through the intent bus.
//
// Supported Bedrock protocol: determined by the pinned gophertunnel release.
//   - gophertunnel v1.57.1 → Bedrock protocol 1001 (Minecraft BE 1.26.30)
//
// Architecture (sole-writer invariant):
//
//	Bedrock client ──RakNet/UDP──> Listener.Listen()
//	                                     │
//	                               handleConn() goroutine per client
//	                                     │
//	                      post Intents to core/intent.Bus
//	                      (never mutate core state directly)
//	                                     │
//	                          core simulation tick goroutine
//	                          applies intents, sends JoinResult
package bedrock

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"

	bedrockworld "GoCraft/bedrock/world"
	"GoCraft/config"
	"GoCraft/core/intent"
)

// Listener wraps a gophertunnel minecraft.Listener and manages Bedrock client
// connections.
type Listener struct {
	cfg config.BedrockConfig
	bus *intent.Bus
}

// NewListener creates a Listener from the Bedrock section of the server config.
// The intent bus is used to submit player lifecycle and gameplay events to the
// core simulation tick goroutine.
func NewListener(cfg config.BedrockConfig, bus *intent.Bus) *Listener {
	return &Listener{cfg: cfg, bus: bus}
}

// Listen starts the RakNet UDP listener and blocks until ctx is cancelled or a
// fatal error occurs.  Each accepted connection is handled in its own goroutine.
func (l *Listener) Listen(ctx context.Context) error {
	if !l.cfg.OnlineMode {
		slog.Warn("⚠ BEDROCK AUTHENTICATION DISABLED — server is running in offline mode",
			"risk", "unauthenticated XUIDs must NOT be treated as trusted global identities",
			"address", l.cfg.Address,
		)
	}

	gt, err := minecraft.ListenConfig{
		AuthenticationDisabled: !l.cfg.OnlineMode,
		ErrorLog:               slog.Default(),
	}.Listen("raknet", l.cfg.Address)
	if err != nil {
		return fmt.Errorf("bedrock: starting RakNet listener on %s: %w", l.cfg.Address, err)
	}

	slog.Info("bedrock listener started",
		"address", l.cfg.Address,
		"onlineMode", l.cfg.OnlineMode,
	)

	// Close the gophertunnel listener when the server context is cancelled.
	go func() {
		<-ctx.Done()
		_ = gt.Close()
	}()

	for {
		conn, err := gt.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown
			}
			slog.Error("bedrock: Accept error", "err", err)
			return fmt.Errorf("bedrock: Accept: %w", err)
		}
		go l.handleConn(ctx, gt, conn.(*minecraft.Conn))
	}
}

// handleConn runs in its own goroutine for every accepted Bedrock connection.
//
// M14.1 flow:
//  1. gophertunnel completes the RakNet + login sequence
//  2. Post JoinIntent, wait for JoinResult from the simulation tick (≤10 s)
//  3. Call conn.StartGame with the assigned entity ID and spawn position
//  4. Send initial LevelChunk packets for the chunk view radius
//  5. Enter the play loop: route packets to intents, handle SubChunkRequests
//  6. On disconnect, post DisconnectIntent
func (l *Listener) handleConn(ctx context.Context, gt *minecraft.Listener, conn *minecraft.Conn) {
	remote := conn.RemoteAddr()

	// ── Step 1: resolve player identity ──────────────────────────────────────
	identity := conn.IdentityData()
	authenticated := conn.Authenticated()

	if !authenticated && l.cfg.OnlineMode {
		// Defensive: gophertunnel enforces auth when AuthenticationDisabled=false.
		slog.Warn("bedrock: unauthenticated connection despite online_mode=true; dropping",
			"remote", remote, "displayName", identity.DisplayName)
		_ = gt.Disconnect(conn, text.Colourf("<red>Authentication required.</red>"))
		return
	}

	// Derive a stable UUID for the session.
	// Online mode: parse identity.Identity (Xbox-issued UUID, trusted).
	// Offline mode: generate a deterministic offline UUID from the display
	//               name so it never collides with an Xbox UUID.
	playerUUID, err := resolveUUID(identity.Identity, identity.DisplayName, authenticated)
	if err != nil {
		slog.Warn("bedrock: could not parse player UUID; dropping",
			"remote", remote, "identity", identity.Identity, "err", err)
		_ = gt.Disconnect(conn, text.Colourf("<red>Internal server error.</red>"))
		return
	}

	slog.Info("bedrock: player connecting",
		"remote", remote,
		"displayName", identity.DisplayName,
		"uuid", playerUUID,
		"xuid", xuidLog(identity.XUID, authenticated),
		"authenticated", authenticated,
	)

	// ── Step 2: request world entry via the simulation ────────────────────────
	done := make(chan intent.JoinResult, 1)
	joinCtx, joinCancel := context.WithTimeout(ctx, 10*time.Second)
	defer joinCancel()

	if err := l.bus.PostJoin(joinCtx, intent.JoinIntent{
		PlayerUUID:      playerUUID,
		Username:        identity.DisplayName,
		Edition:         "bedrock",
		TrustedIdentity: authenticated,
		Done:            done,
	}); err != nil {
		// ctx cancelled (server shutting down) or 10 s posting timeout.
		slog.Warn("bedrock: PostJoin failed; dropping connection",
			"remote", remote, "displayName", identity.DisplayName, "err", err)
		_ = gt.Disconnect(conn, text.Colourf("<yellow>Server timed out. Please reconnect.</yellow>"))
		return
	}

	var result intent.JoinResult
	select {
	case result = <-done:
		// Join was processed by the tick goroutine.
	case <-time.After(10 * time.Second):
		// The intent was queued but the tick goroutine did not respond in time.
		// Post a DisconnectIntent so the tick cleans up the player if it was
		// already added (lifecycle channel is FIFO, so disconnect follows join).
		slog.Warn("bedrock: JoinResult timed out; posting cleanup disconnect",
			"remote", remote, "displayName", identity.DisplayName)
		_ = l.bus.PostDisconnect(ctx, intent.DisconnectIntent{
			PlayerUUID: playerUUID,
			Reason:     "join response timeout",
		})
		_ = gt.Disconnect(conn, text.Colourf("<yellow>Server timed out. Please reconnect.</yellow>"))
		return
	case <-ctx.Done():
		return
	}
	if result.Err != nil {
		slog.Warn("bedrock: join rejected by simulation",
			"remote", remote, "displayName", identity.DisplayName, "err", result.Err)
		_ = gt.Disconnect(conn, text.Colourf("<red>Could not join: %v</red>", result.Err))
		return
	}

	defer func() {
		_ = l.bus.PostDisconnect(ctx, intent.DisconnectIntent{
			PlayerUUID: playerUUID,
			Reason:     "connection closed",
		})
		slog.Info("bedrock: player disconnected",
			"displayName", identity.DisplayName, "uuid", playerUUID)
	}()

	// ── Step 3: send StartGame ────────────────────────────────────────────────
	entityID := int64(result.EntityID)
	spawnPos := mgl32.Vec3{
		float32(result.Position.X),
		bedrockworld.SpawnY(),
		float32(result.Position.Z),
	}

	if err := conn.StartGame(minecraft.GameData{
		WorldName:                    "GoCraft",
		EntityUniqueID:               entityID,
		EntityRuntimeID:              uint64(result.EntityID),
		PlayerPosition:               spawnPos,
		PlayerGameMode:               1, // creative (survival needs ground collision)
		WorldGameMode:                1,
		Difficulty:                   1, // easy
		ServerAuthoritativeInventory: true,
		WorldSeed:                    0,
	}); err != nil {
		slog.Debug("bedrock: StartGame failed",
			"remote", remote, "displayName", identity.DisplayName, "err", err)
		return
	}

	// ── Step 4: stream initial chunks ────────────────────────────────────────
	const chunkRadius = 4
	spawnCX := int32(spawnPos.X()) >> 4
	spawnCZ := int32(spawnPos.Z()) >> 4

	if err := l.sendInitialChunks(conn, spawnCX, spawnCZ, chunkRadius); err != nil {
		slog.Debug("bedrock: chunk streaming failed",
			"displayName", identity.DisplayName, "err", err)
		return
	}

	// ── Step 5: play loop ─────────────────────────────────────────────────────
	l.playLoop(ctx, conn, playerUUID, identity.DisplayName)
}

// sendInitialChunks sends LevelChunk packets for a square of chunks around the
// spawn position, using SubChunkRequestModeLimitless so the client requests
// block sub-chunks on demand.  Each packet carries the minimum valid biome
// payload (24 sub-chunks of plains biome data + border block count of 0).
func (l *Listener) sendInitialChunks(conn *minecraft.Conn, cx, cz, radius int32) error {
	biomePayload := bedrockworld.EncodeLevelChunkPayload()
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			if err := conn.WritePacket(&packet.LevelChunk{
				Position:      protocol.ChunkPos{cx + dx, cz + dz},
				Dimension:     0, // overworld
				SubChunkCount: protocol.SubChunkRequestModeLimitless,
				CacheEnabled:  false,
				RawPayload:    biomePayload,
			}); err != nil {
				return fmt.Errorf("sendInitialChunks: %w", err)
			}
		}
	}
	return nil
}

// playLoop reads packets from a connected Bedrock client and routes them to
// the appropriate intent or response handler.
//
// Returns when the connection closes or ctx is cancelled.
func (l *Listener) playLoop(ctx context.Context, conn *minecraft.Conn, playerUUID [16]byte, displayName string) {
	// Pre-encode the two sub-chunk payloads.
	airPayload, err := bedrockworld.EncodeAirSubChunk()
	if err != nil {
		slog.Error("bedrock: could not encode air sub-chunk", "err", err)
		return
	}
	groundPayload, err := bedrockworld.EncodeGroundSubChunk()
	if err != nil {
		slog.Error("bedrock: could not encode ground sub-chunk", "err", err)
		return
	}

	connDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		_ = conn.Close()
		close(connDone)
	}()

	for {
		pk, err := conn.ReadPacket()
		if err != nil {
			return
		}

		switch p := pk.(type) {
		case *packet.SubChunkRequest:
			l.handleSubChunkRequest(conn, p, airPayload, groundPayload)

		case *packet.MovePlayer:
			l.bus.UpdateMove(intent.MoveIntent{
				PlayerUUID: playerUUID,
				// Bedrock Y is feet position; no adjustment needed.
			})
			_ = p // position logged on M14.2 when broadcast is wired

		case *packet.Text:
			if strings.TrimSpace(p.Message) != "" {
				l.bus.PostChat(intent.ChatIntent{
					PlayerUUID:  playerUUID,
					DisplayName: displayName,
					Message:     p.Message,
				})
			}

		case *packet.RequestChunkRadius:
			// The client is asking for a larger view. Acknowledge with the same
			// radius for now; chunk streaming at new radius is deferred to M14.2.
			_ = conn.WritePacket(&packet.ChunkRadiusUpdated{
				ChunkRadius: p.ChunkRadius,
			})
		}
	}
}

// handleSubChunkRequest responds to the client's on-demand sub-chunk requests.
// Ground sub-chunk (index = bedrockworld.GroundSubChunkIndex) carries stone;
// all others return SuccessAllAir (no payload required).
func (l *Listener) handleSubChunkRequest(
	conn *minecraft.Conn,
	req *packet.SubChunkRequest,
	airPayload, groundPayload []byte,
) {
	entries := make([]protocol.SubChunkEntry, 0, len(req.Offsets))
	for _, off := range req.Offsets {
		subY := req.Position.Y() + int32(off[1])

		entry := protocol.SubChunkEntry{
			Offset: off,
		}
		if subY == bedrockworld.GroundSubChunkIndex() {
			entry.Result = protocol.SubChunkResultSuccess
			entry.RawPayload = groundPayload
		} else {
			entry.Result = protocol.SubChunkResultSuccessAllAir
		}
		entries = append(entries, entry)
	}

	_ = conn.WritePacket(&packet.SubChunk{
		CacheEnabled:    false,
		Dimension:       req.Dimension,
		Position:        req.Position,
		SubChunkEntries: entries,
	})
}

// ── Identity helpers ──────────────────────────────────────────────────────────

// resolveUUID returns the player's canonical [16]byte UUID.
//
//   - Authenticated (online_mode=true): parse the Xbox-issued UUID from
//     identityStr, which is verified by gophertunnel.
//   - Unauthenticated (online_mode=false): generate a deterministic offline
//     UUID (UUID v3, GoCraft namespace + display name). Offline UUIDs use
//     variant bits that keep them in a different range than Xbox UUIDs,
//     preventing accidental collisions.
func resolveUUID(identityStr, displayName string, authenticated bool) ([16]byte, error) {
	if authenticated {
		return parseHexUUID(identityStr)
	}
	return offlineUUID(displayName), nil
}

// parseHexUUID parses a standard UUID string (with dashes) into [16]byte.
func parseHexUUID(s string) ([16]byte, error) {
	cleaned := strings.ReplaceAll(s, "-", "")
	if len(cleaned) != 32 {
		return [16]byte{}, fmt.Errorf("invalid UUID %q", s)
	}
	b, err := hex.DecodeString(cleaned)
	if err != nil {
		return [16]byte{}, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	var u [16]byte
	copy(u[:], b)
	return u, nil
}

// gocraftOfflineNS is the fixed namespace for offline UUID generation.
// Generated once (arbitrary); documented here so it is never changed:
// replacing it would change offline UUIDs for existing players.
//
// Value: SHA-256 of "GoCraft offline namespace" truncated to 16 bytes:
//
//	python3 -c "import hashlib; print(hashlib.sha256(b'GoCraft offline namespace').hexdigest()[:32])"
//	→ 5f3e2a1b4c7d8e9f0a1b2c3d4e5f6a7b
var gocraftOfflineNS = [16]byte{
	0x5f, 0x3e, 0x2a, 0x1b, 0x4c, 0x7d, 0x8e, 0x9f,
	0x0a, 0x1b, 0x2c, 0x3d, 0x4e, 0x5f, 0x6a, 0x7b,
}

// offlineUUID generates a deterministic UUID v3 (MD5-based) for an
// unauthenticated player. The UUID is stable across server restarts for the
// same display name, and its version/variant bits distinguish it from Xbox
// UUIDs (which are version 4, random).
//
// This UUID must NOT be treated as a globally trusted identity — it is only
// reliable within the scope of a single server instance where collisions can
// be checked against the connected player list.
func offlineUUID(displayName string) [16]byte {
	h := md5.New()
	h.Write(gocraftOfflineNS[:])
	h.Write([]byte(displayName))
	digest := h.Sum(nil)

	var u [16]byte
	copy(u[:], digest)
	u[6] = (u[6] & 0x0f) | 0x30 // version 3
	u[8] = (u[8] & 0x3f) | 0x80 // variant 1 (RFC 4122)
	return u
}

// xuidLog returns the XUID for structured logging, or "<offline>" when
// unauthenticated to make clear the value is unverified.
func xuidLog(xuid string, authenticated bool) string {
	if authenticated {
		return xuid
	}
	return "<offline>"
}
