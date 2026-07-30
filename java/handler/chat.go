package handler

// Chat handling for Milestone 7.
//
// Receives C→S Chat Message and Chat Command packets, formats them, and
// broadcasts them as S→C System Chat Message to all online sessions.
// Messages starting with '/' (or arriving via Chat Command) are dispatched
// to the built-in command dispatcher.
//
// All packet IDs are estimates for 1.21.4 (protocol 769); verify against a
// packet capture if the client never shows chat or commands don't fire.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"

	"GoCraft/core/player"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

// ── Chat packet IDs ───────────────────────────────────────────────────────────

const (
	// packetIDChatMessage (C→S) — player sends a chat message.
	// Estimate for 1.21.4; verify against packet dump.
	packetIDChatMessage = 0x06

	// packetIDChatCommand (C→S) — player sends a slash command.
	// The command string arrives without the leading '/'.
	// Estimate for 1.21.4; verify against packet dump.
	packetIDChatCommand = 0x05

	// packetIDSystemChatMessage (S→C) — server sends a chat or system message.
	// Used for both broadcast chat and per-player command feedback.
	// Estimate for 1.21.4; verify against packet dump.
	packetIDSystemChatMessage = 0x6D
)

// ── Dispatch ──────────────────────────────────────────────────────────────────

// handleChatPacket dispatches an incoming chat or command packet.
// Called from the play loop after handlePlayPacket for packets that need
// access to the session manager.
func handleChatPacket(pkt *protocol.Packet, p *player.Player, mgr *session.Manager) error {
	switch pkt.ID {
	case packetIDChatMessage:
		return handleChatMessage(pkt, p, mgr)
	case packetIDChatCommand:
		return handleChatCommand(pkt, p, mgr)
	}
	return nil
}

// handleChatMessage parses C→S Chat Message and either broadcasts the text or
// dispatches it as a command if the player typed a '/' prefix in chat.
//
// Wire layout (1.21.4, fields after Message are ignored in M7):
//
//	String  message (≤ 256 chars)
//	Long    timestamp
//	Long    salt
//	Bool    has_signature
//	Byte[]  signature (256 bytes, only if has_signature)
//	VarInt  message_count
//	Fixed BitSet (20 bits) acknowledged
func handleChatMessage(pkt *protocol.Packet, p *player.Player, mgr *session.Manager) error {
	r := pkt.Reader()
	msg, err := protocol.ReadString(r)
	if err != nil {
		return fmt.Errorf("reading chat message string: %w", err)
	}
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return nil
	}

	// Some clients still prefix commands with '/' in Chat Message even though
	// 1.19+ has a dedicated Chat Command packet. Handle both paths.
	if strings.HasPrefix(msg, "/") {
		slog.Info("command (via chat)", "player", p.Username, "input", msg)
		return dispatchCommand(strings.TrimPrefix(msg, "/"), p, mgr)
	}

	text := fmt.Sprintf("<%s> %s", p.Username, msg)
	slog.Info("chat", "player", p.Username, "message", msg)
	broadcastSystemMessage(mgr, text)
	return nil
}

// handleChatCommand parses C→S Chat Command (command without leading '/').
//
// Wire layout (1.21.4, fields after Command are ignored in M7):
//
//	String  command (no leading '/')
//	Long    timestamp
//	Long    salt
//	VarInt  argument_count
//	…       argument signatures × argument_count (ignored)
//	VarInt  message_count
//	Fixed BitSet (20 bits) acknowledged
func handleChatCommand(pkt *protocol.Packet, p *player.Player, mgr *session.Manager) error {
	r := pkt.Reader()
	cmd, err := protocol.ReadString(r)
	if err != nil {
		return fmt.Errorf("reading chat command string: %w", err)
	}
	slog.Info("command", "player", p.Username, "command", cmd)
	return dispatchCommand(cmd, p, mgr)
}

// ── Command dispatcher ────────────────────────────────────────────────────────

// dispatchCommand parses and executes a server-side command.
// cmd is the command line without the leading '/'.
// M12 will replace this with a proper command tree and Commands packet.
func dispatchCommand(cmd string, p *player.Player, mgr *session.Manager) error {
	parts := strings.Fields(strings.TrimSpace(cmd))
	if len(parts) == 0 {
		return nil
	}

	sess, ok := mgr.Get(p.UUID)
	if !ok {
		return nil // player already disconnected
	}

	switch strings.ToLower(parts[0]) {
	case "help", "?":
		_ = sendSystemMessage(sess.Conn, "Commands: /help, /list")

	case "list":
		var names []string
		mgr.ForEach(func(s *session.Session) {
			names = append(names, s.Player.Username)
		})
		_ = sendSystemMessage(sess.Conn,
			fmt.Sprintf("Online (%d): %s", len(names), strings.Join(names, ", ")))

	default:
		_ = sendSystemMessage(sess.Conn,
			fmt.Sprintf("Unknown command: /%s", parts[0]))
	}
	return nil
}

// ── Send helpers ──────────────────────────────────────────────────────────────

// broadcastSystemMessage sends a plain-text System Chat Message to every
// currently-online session.
func broadcastSystemMessage(mgr *session.Manager, text string) {
	pkt := buildSystemChatMessage(text, false)
	mgr.ForEach(func(s *session.Session) {
		_ = s.Conn.WritePacket(pkt)
	})
}

// sendSystemMessage sends a plain-text System Chat Message to one connection.
func sendSystemMessage(conn *network.ClientConn, text string) error {
	return conn.WritePacket(buildSystemChatMessage(text, false))
}

// buildSystemChatMessage constructs a System Chat Message (S→C) packet.
//
// Wire layout (1.21.4):
//
//	Text Component (NBT)  content
//	Boolean               overlay  (false = chat box, true = action bar)
func buildSystemChatMessage(text string, overlay bool) *protocol.Packet {
	return protocol.NewBuilder(packetIDSystemChatMessage).
		Bytes(nbtTextComponent(text)).
		Bool(overlay).
		Build()
}

// ── Text component NBT encoder ────────────────────────────────────────────────

// nbtTextComponent encodes a plain text string as a minimal Network NBT text
// component, as required by System Chat Message and other packets in 1.20.3+.
//
// Encoding: a root TAG_Compound (no name — network NBT format) containing a
// single TAG_String field named "text", followed by TAG_End.
//
//	0x0A              TAG_Compound root (no name)
//	0x08 "text" …    TAG_String: name="text", value=text
//	0x00              TAG_End
func nbtTextComponent(text string) []byte {
	var buf bytes.Buffer

	buf.WriteByte(0x0A) // TAG_Compound root

	// TAG_String entry: type + name + value
	buf.WriteByte(0x08) // TAG_String

	nameBytes := []byte("text")
	var nameLen [2]byte
	binary.BigEndian.PutUint16(nameLen[:], uint16(len(nameBytes)))
	buf.Write(nameLen[:])
	buf.Write(nameBytes)

	textBytes := []byte(text)
	var textLen [2]byte
	binary.BigEndian.PutUint16(textLen[:], uint16(len(textBytes)))
	buf.Write(textLen[:])
	buf.Write(textBytes)

	buf.WriteByte(0x00) // TAG_End
	return buf.Bytes()
}
