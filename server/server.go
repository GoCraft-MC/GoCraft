// Package server wires together the game core, Java and Bedrock network
// listeners, configuration, and protocol handlers into a runnable GoCraft
// server.
//
// Architecture:
//
//	server.Server
//	  ├─ core/game.Game          — edition-agnostic player registry
//	  ├─ core/intent.Queue       — simulation command bus (M14.1+)
//	  ├─ java adapter            — TCP listener + Java protocol handlers
//	  │    ├─ java/network       — ClientConn, Listener
//	  │    └─ java/handler       — Handshake, Status, Login, Config, Play
//	  └─ bedrock adapter         — RakNet/UDP listener via gophertunnel
//	       └─ bedrock.Listener   — M14.0: accept + auth; M14.1+: play loop
package server

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"GoCraft/bedrock"
	"GoCraft/config"
	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/auth"
	"GoCraft/java/handler"
	"GoCraft/java/network"
	"GoCraft/java/registry"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
	"GoCraft/java/world/anvil"
)

// Server owns the game core and both Java and Bedrock network listeners.
type Server struct {
	cfg  *config.Config
	game *game.Game

	// Java adapter resources
	listener *network.Listener

	// RSA keypair — generated once at startup, shared across all connections.
	privKey   *rsa.PrivateKey
	pubKeyDER []byte

	loginHandler *handler.LoginHandler

	// World and Java encoding resources.
	world        *coreworld.World
	regProvider  registry.Provider
	chunkSender  *javaworld.Sender
	sessions     *session.Manager
	cmds         *handler.Dispatcher

	// Bedrock adapter (nil when bedrock.enabled = false).
	bedrockListener *bedrock.Listener

	// intentBus is the cross-adapter simulation command bus.
	// Both Java (M14.1+) and Bedrock handlers post intents here; the tick
	// goroutine drains and applies them once per tick.
	intentBus *intent.Bus

	// connCount tracks the number of active TCP connections (Java).
	connCount atomic.Int64
}

// New creates a Server with the given configuration.
// It initialises the game core and generates the RSA keypair for online-mode auth.
func New(cfg *config.Config) (*Server, error) {
	privKey, err := auth.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("server: generating RSA keypair: %w", err)
	}
	pubKeyDER, err := auth.MarshalPublicKeyDER(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("server: marshalling public key: %w", err)
	}

	// Open Anvil persistence when WorldDir is configured; fall back to a
	// generation-only flat world otherwise.
	var storage coreworld.Storage
	if cfg.WorldDir != "" {
		st, err := anvil.NewStorage(cfg.WorldDir)
		if err != nil {
			slog.Warn("server: could not open Anvil storage; running without persistence",
				"worldDir", cfg.WorldDir, "err", err)
		} else {
			storage = st
			slog.Info("server: opened Anvil world", "worldDir", cfg.WorldDir)
		}
	}

	cmds := handler.NewDispatcher()
	handler.RegisterBuiltins(cmds)

	bus := intent.NewBus(64, 512)

	s := &Server{
		cfg:         cfg,
		game:        game.New(),
		privKey:     privKey,
		pubKeyDER:   pubKeyDER,
		world:       coreworld.New(&coreworld.FlatGenerator{}, storage),
		regProvider: &registry.VanillaProvider{},
		chunkSender: javaworld.DefaultSender,
		sessions:    session.NewManager(),
		cmds:        cmds,
		intentBus:   bus,
	}
	s.loginHandler = handler.NewLoginHandler(cfg, privKey, pubKeyDER)
	s.listener = network.NewListener(cfg.Addr(), s.handleConn)

	if cfg.Bedrock.Enabled {
		s.bedrockListener = bedrock.NewListener(cfg.Bedrock, bus)
	}
	return s, nil
}

// Run starts the server and blocks until ctx is cancelled or a fatal error occurs.
// All background goroutines are tracked with a WaitGroup and are joined before
// the world is flushed to disk, ensuring clean shutdown of both listeners.
func (s *Server) Run(ctx context.Context) error {
	if s.cfg.JavaEnabled {
		slog.Info("java listener enabled",
			"addr", s.cfg.Addr(),
			"version", s.cfg.VersionName,
			"protocol", s.cfg.ProtocolVersion,
			"onlineMode", s.cfg.OnlineMode,
		)
	}
	if s.cfg.Bedrock.Enabled {
		slog.Info("bedrock listener enabled",
			"addr", s.cfg.Bedrock.Address,
			"onlineMode", s.cfg.Bedrock.OnlineMode,
		)
	}
	slog.Info("starting GoCraft server", "motd", s.cfg.MOTD)

	// Spawn a small set of passive mobs near the world spawn for testing.
	s.spawnTestMobs()

	var wg sync.WaitGroup

	// Entity tick + intent processing at 20 TPS.
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.runEntityTick(ctx)
	}()

	// Bedrock UDP listener (when enabled).
	if s.bedrockListener != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.bedrockListener.Listen(ctx); err != nil {
				slog.Error("bedrock listener stopped with error", "err", err)
			}
		}()
	}

	// Java TCP listener on the main goroutine, or block on ctx if disabled.
	var listenErr error
	if s.cfg.JavaEnabled {
		listenErr = s.listener.Listen(ctx)
	} else {
		<-ctx.Done()
	}

	// ctx is now done: wait for entity tick and Bedrock listener to finish.
	wg.Wait()

	// Flush world to disk regardless of shutdown cause.
	if closeErr := s.world.Close(); closeErr != nil {
		slog.Warn("server: error flushing world on shutdown", "err", closeErr)
	}
	return listenErr
}

// runEntityTick fires tickEntities and tickIntents at 20 TPS until ctx is done.
func (s *Server) runEntityTick(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickIntents()
			s.tickEntities()
		}
	}
}

// tickIntents drains the intent bus and applies each intent to world/player state.
// This is the sole point of mutating player state from adapter goroutines.
func (s *Server) tickIntents() {
	dr := s.intentBus.Drain()

	for _, l := range dr.Lifecycle {
		switch i := l.(type) {
		case intent.JoinIntent:
			s.applyJoin(i)
		case intent.DisconnectIntent:
			s.applyDisconnect(i)
		}
	}

	for _, m := range dr.Moves {
		s.applyMove(m)
	}

	for _, g := range dr.Gameplay {
		switch i := g.(type) {
		case intent.ChatIntent:
			s.applyChat(i)
		}
	}
}

// applyJoin creates a canonical Player, registers it with the game core, and
// sends a JoinResult to the waiting adapter goroutine.
func (s *Server) applyJoin(i intent.JoinIntent) {
	edition := player.ClientEditionBedrock
	if i.Edition == "java" {
		edition = player.ClientEditionJava
	}

	p := player.New(i.PlayerUUID, i.Username, edition)
	if err := s.game.AddPlayer(p); err != nil {
		slog.Warn("applyJoin: duplicate player UUID",
			"name", i.Username, "uuid", i.PlayerUUID, "err", err)
		i.Done <- intent.JoinResult{Err: err}
		return
	}

	handler.OnlineCount.Store(int32(s.game.OnlineCount()))
	slog.Info("player joined via intent",
		"name", p.Username, "uuid", p.UUID,
		"edition", i.Edition, "trusted", i.TrustedIdentity,
		"entityID", p.EntityID)

	i.Done <- intent.JoinResult{
		EntityID: p.EntityID,
		Position: spatial.Vec3{X: 0, Y: 65, Z: 0},
	}
}

// applyDisconnect removes a player from the game core and logs the event.
func (s *Server) applyDisconnect(i intent.DisconnectIntent) {
	s.game.RemovePlayer(i.PlayerUUID)
	handler.OnlineCount.Store(int32(s.game.OnlineCount()))
	slog.Info("player disconnected via intent",
		"uuid", i.PlayerUUID, "reason", i.Reason)
}

// applyMove updates a player's position. The player object is looked up from
// the session manager so that broadcast can follow on the same tick.
func (s *Server) applyMove(m intent.MoveIntent) {
	// TODO(M14.2): look up the bedrock session by UUID and broadcast
	// position to other sessions. For M14.1 the move is accepted and dropped.
	_ = m
}

// applyChat broadcasts a chat message to all active Java sessions.
func (s *Server) applyChat(i intent.ChatIntent) {
	msg := fmt.Sprintf("<%s> %s", i.DisplayName, i.Message)
	handler.BroadcastSystemMessage(s.sessions, msg)
}

// tickEntities advances every registered non-player entity by one game tick:
//   - Gravity is applied when the entity is airborne.
//   - Position is integrated from velocity.
//   - A simple flat-world ground check clamps entities at Y=64.
//   - Dead entities are removed from the manager this tick.
//   - Packets for position updates and despawns are built synchronously, then
//     handed to a goroutine so slow clients cannot stall the simulation.
//
// Ownership: this method is the sole writer of entity spatial/health fields.
// See the concurrency comment on core/entity.Entity for the full invariant.
func (s *Server) tickEntities() {
	start := time.Now()

	const (
		gravity = -0.08 // blocks/tick² downward acceleration
		drag    = 0.98  // horizontal velocity multiplier per tick
		groundY = 64.0  // flat-world surface: top face of Y=63 stone
		minVel  = 1e-6  // below this threshold, zero velocity to avoid float noise
	)

	var (
		moved   []*corentity.Entity // entities whose position changed this tick
		deadIDs []int32             // entity IDs removed from the world this tick
	)

	for _, e := range s.world.Entities.Snapshot() {
		// ── Dead entity cleanup ───────────────────────────────────────────────
		if e.Dead {
			s.world.Entities.Remove(e.EntityID)
			deadIDs = append(deadIDs, e.EntityID)
			slog.Info("entity died", "type", e.Type, "id", e.EntityID)
			continue
		}

		// ── Gravity ───────────────────────────────────────────────────────────
		if !e.OnGround {
			e.VY += gravity
		}

		// ── Position integration ──────────────────────────────────────────────
		prevX, prevY, prevZ := e.Position.X, e.Position.Y, e.Position.Z
		e.Position.X += e.VX
		e.Position.Y += e.VY
		e.Position.Z += e.VZ

		// ── Ground detection (flat-world approximation) ───────────────────────
		if e.Position.Y <= groundY {
			e.Position.Y = groundY
			e.VY = 0
			e.OnGround = true
		} else {
			e.OnGround = false
		}

		// ── Horizontal drag ───────────────────────────────────────────────────
		e.VX *= drag
		e.VZ *= drag
		if math.Abs(e.VX) < minVel {
			e.VX = 0
		}
		if math.Abs(e.VZ) < minVel {
			e.VZ = 0
		}

		// ── Collect moved entities for broadcast ──────────────────────────────
		if e.Position.X != prevX || e.Position.Y != prevY || e.Position.Z != prevZ {
			moved = append(moved, e)
		}
	}

	// Build packets and dispatch network I/O off the tick goroutine.
	// DispatchTickBroadcast reads entity fields here (tick goroutine, sole
	// writer) to build immutable packets before spawning the send goroutine.
	handler.DispatchTickBroadcast(moved, deadIDs, s.sessions)

	// Warn when the CPU work in a tick exceeds the tick budget.
	// Network I/O is off-goroutine and does not count toward this budget.
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		slog.Warn("entity tick overrun", "elapsed", elapsed)
	}
}

// spawnTestMobs populates the world near the spawn point with a handful of
// passive mobs so the entity system can be verified with a vanilla client.
// This is removed or made config-driven in a later milestone.
func (s *Server) spawnTestMobs() {
	type spawn struct {
		t       corentity.EntityType
		x, y, z float64
	}
	mobs := []spawn{
		{corentity.TypeCow, 6, 64, 0},
		{corentity.TypeCow, -6, 64, 4},
		{corentity.TypePig, 0, 64, 7},
		{corentity.TypeSheep, 4, 64, -5},
		{corentity.TypeChicken, -4, 64, -6},
	}
	for _, m := range mobs {
		id := s.game.NextEntityID()
		uuid := newRandomUUID()
		e := corentity.New(id, uuid, m.t, m.x, m.y, m.z)
		e.OnGround = true // spawned on the surface; skip first-tick gravity drop
		s.world.Entities.Add(e)
		slog.Info("spawned entity", "type", m.t, "id", id,
			"x", m.x, "y", m.y, "z", m.z)
	}
}

// newRandomUUID generates a random RFC 4122 version-4 UUID.
func newRandomUUID() [16]byte {
	var uuid [16]byte
	if _, err := cryptorand.Read(uuid[:]); err != nil {
		// crypto/rand failure is extremely rare; panic is acceptable here
		// because the server cannot safely assign unique entity identities.
		panic("server: crypto/rand failed: " + err.Error())
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 1 (RFC 4122)
	return uuid
}

// handleConn is called in its own goroutine for every accepted TCP connection.
func (s *Server) handleConn(conn *network.ClientConn) {
	s.connCount.Add(1)
	defer s.connCount.Add(-1)

	remote := conn.RemoteAddr()

	// ── Handshake ────────────────────────────────────────────────────────────
	_, err := handler.Handshake(conn, s.cfg)
	if err != nil {
		slog.Debug("handshake failed", "remote", remote, "err", err)
		return
	}

	// ── Route by state ───────────────────────────────────────────────────────
	switch conn.State {
	case network.StateStatus:
		if err := handler.HandleStatus(conn, s.cfg); err != nil {
			slog.Debug("status error", "remote", remote, "err", err)
		}

	case network.StateLogin:
		result, err := s.loginHandler.Handle(conn)
		if err != nil {
			slog.Debug("login error", "remote", remote, "err", err)
			return
		}

		// ── Configuration state ──────────────────────────────────────────────
		if err := handler.HandleConfiguration(conn, s.regProvider); err != nil {
			slog.Debug("configuration error", "remote", remote, "err", err)
			return
		}

		// ── Play state ───────────────────────────────────────────────────────
		p := s.registerPlayer(result)
		defer s.game.RemovePlayer(p.UUID)

		if err := handler.HandlePlay(conn, p, s.world, s.chunkSender, s.sessions, s.cmds); err != nil {
			slog.Debug("play error", "remote", remote, "err", err)
		}

	default:
		slog.Warn("unhandled state after handshake", "remote", remote, "state", conn.State)
	}
}

// registerPlayer creates a core Player from a LoginResult, assigns an entity ID
// via the game core, and updates the global online count used in status pings.
func (s *Server) registerPlayer(result *handler.LoginResult) *player.Player {
	// protocol.UUID is [16]byte — convertible to the core's raw [16]byte UUID.
	p := player.New([16]byte(result.UUID), result.Name, player.ClientEditionJava)

	if err := s.game.AddPlayer(p); err != nil {
		// Duplicate UUID — extremely rare; log and continue with assigned ID.
		slog.Warn("duplicate player UUID", "name", p.Username, "uuid", p.UUID, "err", err)
	}

	// Update the status-ping online count.
	handler.OnlineCount.Store(int32(s.game.OnlineCount()))
	slog.Info("player joined", "name", p.Username, "uuid", p.UUID, "entityID", p.EntityID)
	return p
}

// ActiveConns returns the current number of open TCP connections.
func (s *Server) ActiveConns() int64 {
	return s.connCount.Load()
}

// OnlineCount returns the number of players registered with the game core.
func (s *Server) OnlineCount() int {
	return s.game.OnlineCount()
}

// Config returns the server's configuration (read-only).
func (s *Server) Config() *config.Config {
	return s.cfg
}

// Shutdown closes the listener immediately.
func (s *Server) Shutdown() error {
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("server: closing listener: %w", err)
	}
	return nil
}
