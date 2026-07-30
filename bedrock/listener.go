// Package bedrock implements the Minecraft Bedrock Edition network adapter for
// GoCraft.  It accepts UDP/RakNet connections via gophertunnel, authenticates
// players through Xbox Live, and (in later milestones) translates between the
// Bedrock protocol and the edition-agnostic core simulation.
//
// Supported Bedrock protocol: determined by the pinned gophertunnel release.
//   - gophertunnel v1.57.1 → Bedrock protocol 1001 (Minecraft BE 1.26.30)
//
// Architecture:
//
//	Bedrock client ──RakNet/UDP──> Listener.Listen()
//	                                     │
//	                               handleConn() goroutine per client
//	                                     │
//	                          post Intents to core/intent.Queue
//	                          (never mutate core state directly)
package bedrock

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/text"

	"GoCraft/config"
)

// Listener wraps a gophertunnel minecraft.Listener and manages Bedrock client
// connections.
type Listener struct {
	cfg config.BedrockConfig
}

// NewListener creates a Listener from the Bedrock section of the server config.
func NewListener(cfg config.BedrockConfig) *Listener {
	return &Listener{cfg: cfg}
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
			// Accept returns an error when the listener is closed; treat that
			// as a clean shutdown rather than a fatal error.
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("bedrock: Accept error", "err", err)
			return fmt.Errorf("bedrock: Accept: %w", err)
		}
		go l.handleConn(gt, conn.(*minecraft.Conn))
	}
}

// handleConn runs in its own goroutine for every accepted Bedrock connection.
// M14.0 scope: complete the gophertunnel handshake, log identity, then
// disconnect cleanly.  World join is implemented in M14.1.
func (l *Listener) handleConn(gt *minecraft.Listener, conn *minecraft.Conn) {
	remote := conn.RemoteAddr()

	// gophertunnel performs the full Bedrock login sequence (RakNet handshake,
	// resource pack negotiation, Xbox Live auth) inside StartGameContext /
	// DoSpawnContext.  For M14.0 we only need the connection to complete its
	// login phase — we do that by calling StartGame with placeholder data and
	// then immediately disconnecting.
	//
	// M14.1 will replace this with a real StartGame + play loop.
	if err := conn.StartGame(placeholderGameData()); err != nil {
		slog.Debug("bedrock: StartGame failed", "remote", remote, "err", err)
		return
	}

	identity := conn.IdentityData()
	authenticated := conn.Authenticated()

	slog.Info("bedrock: player connected",
		"remote", remote,
		"displayName", identity.DisplayName,
		"xuid", xuidLog(identity.XUID, authenticated),
		"uuid", identity.Identity,
		"authenticated", authenticated,
	)

	if !authenticated && l.cfg.OnlineMode {
		// Should not happen — gophertunnel enforces auth when
		// AuthenticationDisabled is false.  Log defensively and drop.
		slog.Warn("bedrock: unauthenticated connection reached handler despite online_mode=true; dropping",
			"remote", remote, "displayName", identity.DisplayName)
		_ = gt.Disconnect(conn, text.Colourf("<red>Authentication required.</red>"))
		return
	}

	// M14.0: gracefully disconnect — world join not yet implemented.
	_ = gt.Disconnect(conn, text.Colourf(
		"<yellow>Bedrock support coming soon (M14.1).</yellow>",
	))
	slog.Debug("bedrock: disconnected after M14.0 handshake", "remote", remote)
}

// xuidLog returns the XUID for logging, or "<offline>" when unauthenticated.
// Never log an unauthenticated XUID as though it were a verified identity.
func xuidLog(xuid string, authenticated bool) string {
	if authenticated {
		return xuid
	}
	return "<offline>"
}

// placeholderGameData returns minimal GameData so gophertunnel can complete
// the login sequence.  M14.1 will replace this with real world data.
func placeholderGameData() minecraft.GameData {
	return minecraft.GameData{
		WorldName:                    "GoCraft",
		WorldSeed:                    0,
		PlayerPosition:               mgl32.Vec3{0, 65, 0},
		PlayerGameMode:               1, // creative
		WorldGameMode:                1,
		Difficulty:                   1, // easy
		ServerAuthoritativeInventory: true,
	}
}

