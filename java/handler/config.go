package handler

import (
	"bytes"
	"fmt"
	"log/slog"

	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

// ── Configuration state packet IDs ────────────────────────────────────────────

const (
	// Serverbound (C→S) — Configuration state
	packetIDClientInformation     = 0x00
	packetIDServerboundKnownPacks = 0x07
	packetIDAcknowledgeFinish     = 0x03

	// Clientbound (S→C) — Configuration state
	packetIDConfigPluginMessage   = 0x01
	packetIDFinishConfiguration   = 0x03
	packetIDFeatureFlags          = 0x0C
	packetIDUpdateTags            = 0x0D
	packetIDClientboundKnownPacks = 0x0E
)

// HandleConfiguration drives the connection through the Configuration state.
//
// Protocol flow (1.21.4 / protocol 769):
//
//	S→C  Known Packs (0x0E)        — server declares "minecraft:core" v1.21.4
//	C→S  Client Information (0x00) — client settings (may arrive first or after Known Packs)
//	C→S  Known Packs (0x07)        — client confirms cached pack knowledge
//	S→C  Plugin Message (0x01)     — server brand "GoCraft" on "minecraft:brand"
//	S→C  Feature Flags (0x0C)      — enable "minecraft:vanilla" feature set
//	S→C  Update Tags (0x0D)        — empty (client uses cached tags from known pack)
//	S→C  Finish Configuration (0x03)
//	C→S  Acknowledge Finish (0x03)
func HandleConfiguration(conn *network.ClientConn) error {
	// ── Step 1: Advertise known data packs ──────────────────────────────────
	if err := sendKnownPacks(conn); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ── Step 2: Drain client packets until we see Known Packs response ──────
	// The client may send Client Information before or after replying to Known Packs.
	if err := readUntilKnownPacks(conn); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ── Step 3: Server brand ─────────────────────────────────────────────────
	if err := sendConfigPluginMessage(conn, "minecraft:brand", "GoCraft"); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ── Step 4: Feature flags — vanilla feature set ───────────────────────────
	if err := sendFeatureFlags(conn); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ── Step 5: Update Tags — empty (client uses cached data from known pack) ─
	if err := sendUpdateTags(conn); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// ── Step 6: Signal end of configuration ──────────────────────────────────
	if err := conn.WritePacket(protocol.NewBuilder(packetIDFinishConfiguration).Build()); err != nil {
		return fmt.Errorf("config: sending finish configuration: %w", err)
	}

	// ── Step 7: Wait for Acknowledge Finish Configuration (0x03 C→S) ─────────
	pkt, err := conn.ReadPacket()
	if err != nil {
		return fmt.Errorf("config: reading acknowledge finish: %w", err)
	}
	if pkt.ID != packetIDAcknowledgeFinish {
		return fmt.Errorf("config: expected 0x03 (AcknowledgeFinish), got 0x%02X", pkt.ID)
	}

	conn.State = network.StatePlay
	slog.Info("configuration complete, entering play state", "remote", conn.RemoteAddr())
	return nil
}

// ── Clientbound helpers ───────────────────────────────────────────────────────

// sendKnownPacks sends the Clientbound Known Packs packet (0x0E S→C).
//
// We advertise a single pack: "minecraft:core" at version "1.21.4".
// A vanilla client will recognise this pack and confirm it has the data cached,
// allowing the server to skip sending redundant Registry Data entries.
//
// Wire layout:
//
//	VarInt   pack_count (1)
//	String   namespace  "minecraft"
//	String   id         "core"
//	String   version    "1.21.4"
func sendKnownPacks(conn *network.ClientConn) error {
	pkt := protocol.NewBuilder(packetIDClientboundKnownPacks).
		VarInt(1).
		String("minecraft").
		String("core").
		String("1.21.4").
		Build()
	return conn.WritePacket(pkt)
}

// sendConfigPluginMessage sends a Plugin Message packet (0x01 S→C) in Configuration state.
//
// The "minecraft:brand" channel expects the brand string encoded as a Minecraft
// string (VarInt length + UTF-8 bytes), which is what WriteString produces.
func sendConfigPluginMessage(conn *network.ClientConn, channel, data string) error {
	// Encode the data as a Minecraft string so the client parses it correctly.
	var dataBuf bytes.Buffer
	if err := protocol.WriteString(&dataBuf, data); err != nil {
		return fmt.Errorf("encoding plugin message data: %w", err)
	}
	pkt := protocol.NewBuilder(packetIDConfigPluginMessage).
		String(channel).
		Bytes(dataBuf.Bytes()).
		Build()
	return conn.WritePacket(pkt)
}

// sendFeatureFlags sends the Feature Flags packet (0x0C S→C).
//
// We declare the single "minecraft:vanilla" flag, which enables the full
// vanilla feature set (required for the client to accept standard gameplay).
func sendFeatureFlags(conn *network.ClientConn) error {
	pkt := protocol.NewBuilder(packetIDFeatureFlags).
		VarInt(1).
		String("minecraft:vanilla").
		Build()
	return conn.WritePacket(pkt)
}

// sendUpdateTags sends the Update Tags packet (0x0D S→C) with zero tag registries.
//
// Because we advertised the "minecraft:core" known pack and the client confirmed
// knowledge of it, the client already has all tag data cached locally.
func sendUpdateTags(conn *network.ClientConn) error {
	pkt := protocol.NewBuilder(packetIDUpdateTags).
		VarInt(0). // registry count = 0
		Build()
	return conn.WritePacket(pkt)
}

// ── Serverbound helpers ───────────────────────────────────────────────────────

// readUntilKnownPacks reads incoming Configuration-state packets, discarding
// Client Information, until the client's Known Packs response (0x07) arrives.
//
// The client may send these in either order:
//
//	C→S Client Information (0x00)  — client display and locale settings
//	C→S Known Packs (0x07)         — acknowledgement of server's known packs
func readUntilKnownPacks(conn *network.ClientConn) error {
	for {
		pkt, err := conn.ReadPacket()
		if err != nil {
			return fmt.Errorf("reading configuration client packet: %w", err)
		}
		switch pkt.ID {
		case packetIDClientInformation:
			slog.Debug("client information received (configuration)", "remote", conn.RemoteAddr())
			// No fields need to be parsed for basic play; just acknowledge receipt.
		case packetIDServerboundKnownPacks:
			slog.Debug("client known packs acknowledged", "remote", conn.RemoteAddr())
			return nil
		default:
			slog.Debug("unexpected configuration packet (ignoring)",
				"remote", conn.RemoteAddr(), "id", fmt.Sprintf("0x%02X", pkt.ID))
		}
	}
}
