package handler

import (
	"bytes"
	"fmt"
	"log/slog"

	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/registry"
)

// HandleConfiguration drives the connection through the Configuration state.
//
// Protocol flow (1.21.4 / protocol 769), matching Mojang's configuration tasks:
//
//	S→C  Plugin Message (0x01)     — server brand "GoCraft" on "minecraft:brand"
//	S→C  Feature Flags (0x0C)      — enable "minecraft:vanilla" feature set
//	S→C  Known Packs (0x0E)        — server declares "minecraft:core" v1.21.4
//	C→S  Client Information (0x00) — client settings (may arrive before Known Packs response)
//	C→S  Known Packs (0x07)        — client confirms cached pack knowledge
//	S→C  Registry Data (0x07)      — all 12 synchronized registries; values from known pack
//	S→C  Update Tags (0x0D)        — complete network-safe tag snapshot using assigned IDs
//	S→C  Finish Configuration (0x03)
//	C→S  Acknowledge Finish (0x03)
func HandleConfiguration(conn *network.ClientConn, reg registry.Provider) error {
	// Mojang sends brand and enabled features before starting the registry
	// synchronization task and Known Packs negotiation.
	if err := sendConfigPluginMessage(conn, "minecraft:brand", "GoCraft"); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := sendFeatureFlags(conn); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if err := sendKnownPacks(conn, reg.Packs()); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	selectedPacks, err := readUntilKnownPacks(conn)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Known Packs permits entry NBT to be omitted, but the complete registry
	// and tag snapshots are still required before the client can freeze them.
	if err := reg.SendRegistries(conn, selectedPacks); err != nil {
		return fmt.Errorf("config: sending registries: %w", err)
	}
	if err := reg.SendTags(conn); err != nil {
		return fmt.Errorf("config: sending tags: %w", err)
	}

	if err := conn.WritePacket(protocol.NewBuilder(packetIDFinishConfiguration).Build()); err != nil {
		return fmt.Errorf("config: sending finish configuration: %w", err)
	}

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
// The packs slice comes from a registry.Provider so the set of advertised packs
// can differ between VanillaProvider (minecraft:core 1.21.4) and future
// ExplicitProvider implementations (custom dimensions, Bedrock translation, etc.).
//
// Wire layout:
//
//	VarInt   pack_count
//	String   namespace
//	String   id
//	String   version
//	  … repeated for each pack …
func sendKnownPacks(conn *network.ClientConn, packs []registry.Pack) error {
	b := protocol.NewBuilder(packetIDClientboundKnownPacks).
		VarInt(int32(len(packs)))
	for _, p := range packs {
		b.String(p.Namespace).String(p.ID).String(p.Version)
	}
	return conn.WritePacket(b.Build())
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

// ── Serverbound helpers ───────────────────────────────────────────────────────

// readUntilKnownPacks reads incoming Configuration-state packets, discarding
// Client Information, until the client's Known Packs response (0x07) arrives.
//
// The client may send these in either order:
//
//	C→S Client Information (0x00)  — client display and locale settings
//	C→S Known Packs (0x07)         — acknowledgement of server's known packs
func readUntilKnownPacks(conn *network.ClientConn) ([]registry.Pack, error) {
	for {
		pkt, err := conn.ReadPacket()
		if err != nil {
			return nil, fmt.Errorf("reading configuration client packet: %w", err)
		}
		switch pkt.ID {
		case packetIDClientInformation:
			slog.Debug("client information received (configuration)", "remote", conn.RemoteAddr())
			// No fields need to be parsed for basic play; just acknowledge receipt.
		case packetIDServerboundKnownPacks:
			packs, err := decodeKnownPacks(pkt.Data)
			if err != nil {
				return nil, fmt.Errorf("decoding Known Packs response: %w", err)
			}
			slog.Info("client known packs acknowledged",
				"remote", conn.RemoteAddr(), "packs", len(packs))
			return packs, nil
		default:
			slog.Debug("unexpected configuration packet (ignoring)",
				"remote", conn.RemoteAddr(), "id", fmt.Sprintf("0x%02X", pkt.ID))
		}
	}
}

func decodeKnownPacks(data []byte) ([]registry.Pack, error) {
	r := bytes.NewReader(data)
	count, err := protocol.ReadVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("reading pack count: %w", err)
	}
	if count < 0 || count > 64 {
		return nil, fmt.Errorf("invalid pack count %d", count)
	}

	packs := make([]registry.Pack, 0, count)
	for i := int32(0); i < count; i++ {
		namespace, err := protocol.ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("pack %d namespace: %w", i, err)
		}
		id, err := protocol.ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("pack %d id: %w", i, err)
		}
		version, err := protocol.ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("pack %d version: %w", i, err)
		}
		packs = append(packs, registry.Pack{Namespace: namespace, ID: id, Version: version})
	}
	if r.Len() != 0 {
		return nil, fmt.Errorf("%d trailing bytes after Known Packs response", r.Len())
	}
	return packs, nil
}
