// Package server wires together the network listener, configuration, and
// protocol handlers into a runnable Minecraft server.
package server

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"GoCraft/auth"
	"GoCraft/config"
	"GoCraft/handler"
	"GoCraft/network"
	"GoCraft/protocol"
)

// Server is the top-level object that owns the listener and dispatches connections.
type Server struct {
	cfg      *config.Config
	listener *network.Listener

	// RSA keypair — generated once at startup, shared across all connections.
	privKey   *rsa.PrivateKey
	pubKeyDER []byte

	// loginHandler is initialised from the keypair and reused for every login.
	loginHandler *handler.LoginHandler

	// connCount tracks the number of active TCP connections.
	connCount atomic.Int64
}

// New creates a Server with the given configuration.
// It generates the RSA keypair needed for online-mode authentication.
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
		if errors.Is(err, handler.ErrLoginNotImplemented) {
			// Fall through — login IS now implemented below.
		} else {
			slog.Debug("handshake failed", "remote", remote, "err", err)
			return
		}
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
		// Login completed, conn is now in StateConfiguration.
		// Configuration state (Milestone 3) is not yet implemented.
		// Send a friendly disconnect so the client shows a readable message.
		s.sendConfigurationDisconnect(conn, result)

	default:
		slog.Warn("unhandled state after handshake", "remote", remote, "state", conn.State)
	}
}

// sendConfigurationDisconnect sends a Disconnect packet in the Configuration
// state so the client displays a message instead of a raw connection drop.
//
// Configuration Disconnect (0x02 S→C):
//
//	String  reason  JSON text component
func (s *Server) sendConfigurationDisconnect(conn *network.ClientConn, result *handler.LoginResult) {
	msg := fmt.Sprintf(
		`{"text":"Welcome, %s! GoCraft is still starting up — Configuration state (Milestone 3) is not yet implemented. Please reconnect later.","color":"yellow"}`,
		result.Name,
	)
	pkt := protocol.NewBuilder(0x02).String(msg).Build()
	if err := conn.WritePacket(pkt); err != nil {
		slog.Debug("configuration disconnect send failed",
			"remote", conn.RemoteAddr(), "err", err)
	}
	slog.Info("player greeted and disconnected (awaiting Milestone 3)",
		"name", result.Name,
		"uuid", result.UUID,
		"remote", conn.RemoteAddr(),
	)
}

// ActiveConns returns the current number of open TCP connections.
func (s *Server) ActiveConns() int64 {
	return s.connCount.Load()
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
