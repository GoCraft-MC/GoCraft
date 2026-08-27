package handler

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"GoCraft/config"
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
func HandleConfiguration(conn *network.ClientConn, reg registry.Provider, rp config.JavaResourcePackConfig) error {
	// Mojang sends brand and enabled features before starting the registry
	// synchronization task and Known Packs negotiation.
	if err := sendConfigPluginMessage(conn, "minecraft:brand", "GoCraft"); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := sendFeatureFlags(conn); err != nil {
		return fmt.Errorf("config: %w", err)
	}

	if rp.Enabled && rp.URL != "" {
		if err := sendResourcePackPush(conn, rp); err != nil {
			return fmt.Errorf("config: resource pack: %w", err)
		}
	}

	if err := sendKnownPacks(conn, reg.Packs()); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	selectedPacks, err := readUntilKnownPacks(conn, rp.Forced)
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
func readUntilKnownPacks(conn *network.ClientConn, packForced bool) ([]registry.Pack, error) {
	for {
		pkt, err := conn.ReadPacket()
		if err != nil {
			return nil, fmt.Errorf("reading configuration client packet: %w", err)
		}
		switch pkt.ID {
		case packetIDClientInformation:
			slog.Debug("client information received (configuration)", "remote", conn.RemoteAddr())
			// No fields need to be parsed for basic play; just acknowledge receipt.
		case packetIDResourcePackResponse:
			// Resource Pack Response: UUID (16 bytes) + status VarInt
			// Status: 1=loaded 2=declined 3=failed 4=accepted
			if len(pkt.Data) >= 17 {
				status, _ := protocol.ReadVarInt(bytes.NewReader(pkt.Data[16:]))
				switch status {
				case 2: // declined
					if packForced {
						return nil, fmt.Errorf("player declined required resource pack")
					}
					slog.Info("player declined resource pack", "remote", conn.RemoteAddr())
				case 3: // failed
					slog.Warn("player failed to download resource pack", "remote", conn.RemoteAddr())
				case 4: // accepted — still downloading, wait for loaded (1)
					slog.Debug("player accepted resource pack", "remote", conn.RemoteAddr())
				case 1: // successfully loaded
					slog.Info("player loaded resource pack", "remote", conn.RemoteAddr())
				}
			}
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

// sendResourcePackPush sends the Resource Pack Push (configuration) packet.
//
// Wire layout (1.21.4):
//
//	UUID      (16 bytes)  — deterministic from URL so repeated joins reuse cache
//	String    URL
//	String    Hash        — SHA-1 hex of the zip, or ""
//	Boolean   Forced      — kick if declined
//	Boolean   Has prompt
//	NBT       Prompt      — optional MiniMessage text shown before the dialog
func sendResourcePackPush(conn *network.ClientConn, rp config.JavaResourcePackConfig) error {
	uuid := urlToPackUUID(rp.URL)
	hash := strings.ToLower(strings.TrimSpace(rp.Hash))

	b := protocol.NewBuilder(packetIDResourcePackPush)
	b.Bytes(uuid[:])
	b.String(rp.URL)
	b.String(hash)
	b.Bool(rp.Forced)
	if rp.Prompt != "" {
		b.Bool(true)
		b.Bytes(nbtTextComponent(ParseMiniMessage(rp.Prompt)))
	} else {
		b.Bool(false)
	}
	return conn.WritePacket(b.Build())
}

// urlToPackUUID derives a stable UUID (version 5 / SHA-1 namespace) from a URL
// so the same pack URL always maps to the same UUID — clients skip re-downloading
// a pack they already have cached from a previous session.
func urlToPackUUID(url string) [16]byte {
	h := sha1.Sum([]byte("gocraft:resource_pack:" + url))
	var uuid [16]byte
	copy(uuid[:], h[:16])
	// Set version 5 (SHA-1 name-based) and variant bits.
	uuid[6] = (uuid[6] & 0x0F) | 0x50
	uuid[8] = (uuid[8] & 0x3F) | 0x80
	return uuid
}

// keep compiler happy for hex import used in hash validation elsewhere
var _ = hex.EncodeToString
var _ = binary.BigEndian
