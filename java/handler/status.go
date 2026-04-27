package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"

	"GoCraft/config"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

// OnlineCount is the global player count used in status responses.
// Updated atomically by the game core when players join or leave.
var OnlineCount atomic.Int32

// statusVersion mirrors the "version" field in the status JSON.
type statusVersion struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

// statusPlayers mirrors the "players" field in the status JSON.
type statusPlayers struct {
	Max    int            `json:"max"`
	Online int            `json:"online"`
	Sample []statusSample `json:"sample"`
}

// statusSample is one entry in the player sample list shown on the multiplayer screen.
type statusSample struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// statusDescription holds the MOTD component.
type statusDescription struct {
	Text string `json:"text"`
}

// statusResponse is the top-level JSON payload for a 0x00 Status Response packet.
// Its shape matches what the vanilla Minecraft client expects.
type statusResponse struct {
	Version     statusVersion     `json:"version"`
	Players     statusPlayers     `json:"players"`
	Description statusDescription `json:"description"`
	// Favicon is an optional "data:image/png;base64,<data>" string.
	// Omit the field entirely when empty so the client uses a default icon.
	Favicon string `json:"favicon,omitempty"`
}

// HandleStatus serves one complete server-list ping exchange:
//
//  1. Read Status Request (0x00, empty)
//  2. Send Status Response (0x00, JSON)
//  3. Read Ping Request   (0x01, int64)
//  4. Send Pong Response  (0x01, same int64)
//
// Either the ping or the connection close is acceptable after step 2;
// some clients do not send a ping.
func HandleStatus(conn *network.ClientConn, cfg *config.Config) error {
	// --- Step 1: Status Request ---
	pkt, err := conn.ReadPacket()
	if err != nil {
		return fmt.Errorf("status: reading request: %w", err)
	}
	if pkt.ID != packetIDStatusRequest {
		return fmt.Errorf("status: expected packet 0x00, got 0x%02X", pkt.ID)
	}

	// --- Step 2: Status Response ---
	payload, err := buildStatusJSON(cfg)
	if err != nil {
		return fmt.Errorf("status: building response JSON: %w", err)
	}

	responsePkt := protocol.NewBuilder(packetIDStatusResponse).
		String(string(payload)).
		Build()

	if err := conn.WritePacket(responsePkt); err != nil {
		return fmt.Errorf("status: writing response: %w", err)
	}

	slog.Info("status ping served", "remote", conn.RemoteAddr())

	// --- Step 3: Ping Request (optional) ---
	pkt, err = conn.ReadPacket()
	if err != nil {
		// The client closed the connection without sending a ping — that's fine.
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		return fmt.Errorf("status: reading ping: %w", err)
	}
	if pkt.ID != packetIDPingRequest {
		// Unexpected packet; just close.
		return fmt.Errorf("status: expected ping 0x01, got 0x%02X", pkt.ID)
	}

	r := pkt.Reader()
	pingPayload, err := protocol.ReadLong(r)
	if err != nil {
		return fmt.Errorf("status: reading ping payload: %w", err)
	}

	// --- Step 4: Pong Response ---
	pongPkt := protocol.NewBuilder(packetIDPongResponse).
		Long(pingPayload).
		Build()

	if err := conn.WritePacket(pongPkt); err != nil {
		return fmt.Errorf("status: writing pong: %w", err)
	}

	slog.Debug("pong sent", "remote", conn.RemoteAddr(), "payload", pingPayload)
	return nil
}

// buildStatusJSON marshals the status payload using the server configuration.
func buildStatusJSON(cfg *config.Config) ([]byte, error) {
	resp := statusResponse{
		Version: statusVersion{
			Name:     cfg.VersionName,
			Protocol: cfg.ProtocolVersion,
		},
		Players: statusPlayers{
			Max:    cfg.MaxPlayers,
			Online: int(OnlineCount.Load()),
			Sample: []statusSample{},
		},
		Description: statusDescription{
			Text: cfg.MOTD,
		},
	}
	return json.Marshal(resp)
}
