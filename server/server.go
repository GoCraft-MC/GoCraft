// Package server wires together the game core, Java network listener,
// configuration, and protocol handlers into a runnable GoCraft server.
//
// Architecture:
//
//	server.Server
//	  ├─ core/game.Game          — edition-agnostic player registry
//	  └─ java adapter            — TCP listener + Java protocol handlers
//	       ├─ java/network       — ClientConn, Listener
//	       └─ java/handler       — Handshake, Status, Login, Config, Play
package server

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/auth"
	"GoCraft/java/handler"
	"GoCraft/java/network"
	"GoCraft/java/registry"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
	"GoCraft/java/world/anvil"
)

// Server owns the game core and the Java Edition TCP listener.
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

	// connCount tracks the number of active TCP connections.
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

	s := &Server{
		cfg:         cfg,
		game:        game.New(),
		privKey:     privKey,
		pubKeyDER:   pubKeyDER,
		world:       coreworld.New(&coreworld.FlatGenerator{}, storage),
		regProvider: &registry.VanillaProvider{},
		chunkSender: javaworld.DefaultSender,
		sessions:    session.NewManager(),
	}
	s.loginHandler = handler.NewLoginHandler(cfg, privKey, pubKeyDER)
	s.listener = network.NewListener(cfg.Addr(), s.handleConn)
	return s, nil
}

// Run starts the server and blocks until ctx is cancelled or a fatal error occurs.
// When the listener stops, all dirty chunks are flushed to disk before returning.
func (s *Server) Run(ctx context.Context) error {
	slog.Info("starting GoCraft server",
		"addr", s.cfg.Addr(),
		"motd", s.cfg.MOTD,
		"version", s.cfg.VersionName,
		"protocol", s.cfg.ProtocolVersion,
		"onlineMode", s.cfg.OnlineMode,
	)

	// Spawn a small set of passive mobs near the world spawn for testing.
	s.spawnTestMobs()

	// Run the entity tick at 20 TPS alongside the network listener.
	go s.runEntityTick(ctx)

	err := s.listener.Listen(ctx)

	// Flush world to disk regardless of whether Listen returned cleanly.
	if closeErr := s.world.Close(); closeErr != nil {
		slog.Warn("server: error flushing world on shutdown", "err", closeErr)
	}
	return err
}

// runEntityTick fires tickEntities at 20 TPS (every 50 ms) until ctx is done.
func (s *Server) runEntityTick(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickEntities()
		}
	}
}

// tickEntities advances every registered non-player entity by one game tick:
//   - Gravity is applied when the entity is airborne.
//   - Position is integrated from velocity.
//   - A simple flat-world ground check clamps entities at Y=64.
//   - Dead entities are removed and despawned.
//   - Moving entities broadcast a Teleport Entity update to all sessions.
//
// Horizontal drag is applied each tick so that any future momentum (knockback,
// projectile impact) naturally decays.
func (s *Server) tickEntities() {
	const (
		gravity = -0.08 // blocks/tick² downward acceleration
		drag    = 0.98  // horizontal velocity multiplier per tick
		groundY = 64.0  // flat-world surface: top face of Y=63 stone
		minVel  = 1e-6  // below this magnitude, velocity is zeroed (avoid float noise)
	)

	for _, e := range s.world.Entities.Snapshot() {
		// ── Dead entity cleanup ───────────────────────────────────────────────
		if e.Dead {
			s.world.Entities.Remove(e.EntityID)
			handler.BroadcastRemoveEntity(e.EntityID, s.sessions)
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

		// ── Position broadcast ────────────────────────────────────────────────
		moved := e.Position.X != prevX || e.Position.Y != prevY || e.Position.Z != prevZ
		if moved {
			handler.BroadcastEntityPosition(e, s.sessions)
		}
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

		if err := handler.HandlePlay(conn, p, s.world, s.chunkSender, s.sessions); err != nil {
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
