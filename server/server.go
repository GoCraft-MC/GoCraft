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
	"crypto/rsa"
	"fmt"
	"log/slog"
	"sync/atomic"

	"GoCraft/config"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/java/auth"
	"GoCraft/java/handler"
	"GoCraft/java/network"
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

	s := &Server{
		cfg:       cfg,
		game:      game.New(),
		privKey:   privKey,
		pubKeyDER: pubKeyDER,
	}
	s.loginHandler = handler.NewLoginHandler(cfg, privKey, pubKeyDER)
	s.listener = network.NewListener(cfg.Addr(), s.handleConn)
	return s, nil
}

// Run starts the server and blocks until ctx is cancelled or a fatal error occurs.
func (s *Server) Run(ctx context.Context) error {
	slog.Info("starting GoCraft server",
		"addr", s.cfg.Addr(),
		"motd", s.cfg.MOTD,
		"version", s.cfg.VersionName,
		"protocol", s.cfg.ProtocolVersion,
		"onlineMode", s.cfg.OnlineMode,
	)
	return s.listener.Listen(ctx)
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
		if err := handler.HandleConfiguration(conn); err != nil {
			slog.Debug("configuration error", "remote", remote, "err", err)
			return
		}

		// ── Play state ───────────────────────────────────────────────────────
		p := s.registerPlayer(result)
		defer s.game.RemovePlayer(p.UUID)

		if err := handler.HandlePlay(conn, p); err != nil {
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
