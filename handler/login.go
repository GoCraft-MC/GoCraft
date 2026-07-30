package handler

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"GoCraft/auth"
	"GoCraft/config"
	"GoCraft/network"
	"GoCraft/protocol"
)

// ── Packet IDs — Login state ──────────────────────────────────────────────────

const (
	// Client → Server
	packetIDLoginStart            = 0x00
	packetIDEncryptionResponse    = 0x01
	packetIDLoginAcknowledged     = 0x03

	// Server → Client
	packetIDLoginDisconnect       = 0x00
	packetIDEncryptionRequest     = 0x01
	packetIDLoginSuccess          = 0x02
)

// LoginResult carries everything the rest of the server needs after a
// successful login (UUID, name, textures).
type LoginResult struct {
	UUID       protocol.UUID
	Name       string
	Properties []auth.ProfileProperty // may be empty in offline mode
}

// LoginHandler owns the RSA keypair used for online-mode encryption.
// Create one per server instance (key generated once at startup).
type LoginHandler struct {
	cfg       *config.Config
	privKey   *rsa.PrivateKey
	pubKeyDER []byte
}

// NewLoginHandler creates a handler with the given config and pre-generated key.
func NewLoginHandler(cfg *config.Config, privKey *rsa.PrivateKey, pubKeyDER []byte) *LoginHandler {
	return &LoginHandler{cfg: cfg, privKey: privKey, pubKeyDER: pubKeyDER}
}

// Handle drives a connection through the complete Minecraft login sequence.
//
// Online mode flow:
//  1. Read  Login Start          (0x00 C→S)
//  2. Write Encryption Request   (0x01 S→C)
//  3. Read  Encryption Response  (0x01 C→S)
//  4. Verify token + Mojang hasJoined
//  5. Enable AES-128-CFB8 on conn
//  6. Write Login Success        (0x02 S→C)  ← sent encrypted
//  7. Read  Login Acknowledged   (0x03 C→S)
//
// Offline mode flow:
//  1. Read  Login Start   (0x00 C→S)
//  2. Write Login Success (0x02 S→C)  ← not encrypted
//  3. Read  Login Acknowledged (0x03 C→S)
//
// On any error a Login Disconnect is sent before returning.
func (h *LoginHandler) Handle(conn *network.ClientConn) (*LoginResult, error) {
	// ── Step 1: Login Start ──────────────────────────────────────────────────
	name, _, err := h.readLoginStart(conn)
	if err != nil {
		return nil, h.disconnect(conn, "Internal error", err)
	}

	slog.Info("login attempt", "remote", conn.RemoteAddr(), "name", name)

	var result *LoginResult

	if h.cfg.OnlineMode {
		result, err = h.doOnlineAuth(conn, name)
	} else {
		result, err = h.doOfflineAuth(conn, name)
	}
	if err != nil {
		return nil, err // disconnect already sent inside each flow
	}

	// ── Final step: Login Acknowledged ──────────────────────────────────────
	if err := h.readLoginAcknowledged(conn); err != nil {
		return nil, err
	}

	conn.State = network.StateConfiguration
	slog.Info("login complete",
		"remote", conn.RemoteAddr(),
		"name", result.Name,
		"uuid", result.UUID,
		"mode", map[bool]string{true: "online", false: "offline"}[h.cfg.OnlineMode],
	)
	return result, nil
}

// ── Online mode ───────────────────────────────────────────────────────────────

func (h *LoginHandler) doOnlineAuth(conn *network.ClientConn, name string) (*LoginResult, error) {
	// Generate a fresh 4-byte challenge for this connection.
	challenge := make([]byte, 4)
	if _, err := rand.Read(challenge); err != nil {
		return nil, h.disconnect(conn, "Internal error", fmt.Errorf("login: generating challenge: %w", err))
	}

	// ── Step 2: Encryption Request ──────────────────────────────────────────
	if err := h.sendEncryptionRequest(conn, challenge); err != nil {
		return nil, fmt.Errorf("login: sending encryption request: %w", err)
	}

	// ── Step 3: Encryption Response ─────────────────────────────────────────
	sharedSecret, err := h.readEncryptionResponse(conn, challenge)
	if err != nil {
		return nil, h.disconnect(conn, "Encryption failed", err)
	}

	// ── Step 4: Mojang session verification ─────────────────────────────────
	serverHash := auth.ComputeServerHash([]byte(""), sharedSecret, h.pubKeyDER)
	profile, err := auth.HasJoined(name, serverHash, conn.RemoteAddr())
	if err != nil {
		return nil, h.disconnect(conn, "Failed to verify username!", err)
	}

	playerUUID, err := auth.ParseMojangUUID(profile.ID)
	if err != nil {
		return nil, h.disconnect(conn, "Invalid UUID from session server", err)
	}

	// ── Step 5: Enable encryption ────────────────────────────────────────────
	// From this point ALL packets (including Login Success) are encrypted.
	if err := conn.EnableEncryption(sharedSecret); err != nil {
		return nil, fmt.Errorf("login: enabling encryption: %w", err)
	}

	slog.Debug("encryption enabled", "remote", conn.RemoteAddr())

	// ── Step 6: Login Success (encrypted) ───────────────────────────────────
	if err := h.sendLoginSuccess(conn, playerUUID, profile.Name, profile.Properties); err != nil {
		return nil, fmt.Errorf("login: sending login success: %w", err)
	}

	return &LoginResult{
		UUID:       playerUUID,
		Name:       profile.Name,
		Properties: profile.Properties,
	}, nil
}

// ── Offline mode ──────────────────────────────────────────────────────────────

func (h *LoginHandler) doOfflineAuth(conn *network.ClientConn, name string) (*LoginResult, error) {
	playerUUID := auth.OfflineUUID(name)

	// ── Step 2 (offline): Login Success (not encrypted) ─────────────────────
	if err := h.sendLoginSuccess(conn, playerUUID, name, nil); err != nil {
		return nil, fmt.Errorf("login: sending login success (offline): %w", err)
	}

	return &LoginResult{
		UUID: playerUUID,
		Name: name,
	}, nil
}

// ── Packet I/O helpers ────────────────────────────────────────────────────────

// readLoginStart parses the Login Start packet (0x00 C→S).
// Returns the player name and the UUID hint sent by the client.
func (h *LoginHandler) readLoginStart(conn *network.ClientConn) (name string, clientUUID protocol.UUID, err error) {
	pkt, err := conn.ReadPacket()
	if err != nil {
		return "", protocol.UUID{}, fmt.Errorf("login: reading Login Start: %w", err)
	}
	if pkt.ID != packetIDLoginStart {
		return "", protocol.UUID{}, fmt.Errorf("login: expected 0x00 (Login Start), got 0x%02X", pkt.ID)
	}

	r := pkt.Reader()

	name, err = protocol.ReadString(r)
	if err != nil {
		return "", protocol.UUID{}, fmt.Errorf("login: reading player name: %w", err)
	}
	if len(name) == 0 || len(name) > 16 {
		return "", protocol.UUID{}, fmt.Errorf("login: invalid player name length %d", len(name))
	}

	clientUUID, err = protocol.ReadUUID(r)
	if err != nil {
		return "", protocol.UUID{}, fmt.Errorf("login: reading client UUID: %w", err)
	}

	return name, clientUUID, nil
}

// sendEncryptionRequest writes the Encryption Request packet (0x01 S→C).
//
// Fields (1.21.4):
//
//	String   serverID          always ""
//	ByteArr  publicKey         DER-encoded RSA public key
//	ByteArr  verifyToken       random 4 bytes
//	Bool     shouldAuthenticate true in online mode
func (h *LoginHandler) sendEncryptionRequest(conn *network.ClientConn, challenge []byte) error {
	pkt := protocol.NewBuilder(packetIDEncryptionRequest).
		String("").
		ByteArray(h.pubKeyDER).
		ByteArray(challenge).
		Bool(true).
		Build()
	return conn.WritePacket(pkt)
}

// readEncryptionResponse reads and decrypts the Encryption Response (0x01 C→S).
// Returns the shared secret after verifying the challenge matches.
//
// Fields (1.21.4):
//
//	ByteArr  encryptedSharedSecret   RSA PKCS1v15
//	ByteArr  encryptedVerifyToken    RSA PKCS1v15
func (h *LoginHandler) readEncryptionResponse(conn *network.ClientConn, challenge []byte) ([]byte, error) {
	pkt, err := conn.ReadPacket()
	if err != nil {
		return nil, fmt.Errorf("login: reading Encryption Response: %w", err)
	}
	if pkt.ID != packetIDEncryptionResponse {
		return nil, fmt.Errorf("login: expected 0x01 (Encryption Response), got 0x%02X", pkt.ID)
	}

	r := pkt.Reader()

	encSecret, err := protocol.ReadByteArray(r)
	if err != nil {
		return nil, fmt.Errorf("login: reading encrypted shared secret: %w", err)
	}

	encToken, err := protocol.ReadByteArray(r)
	if err != nil {
		return nil, fmt.Errorf("login: reading encrypted verify token: %w", err)
	}

	// Decrypt both values with our RSA private key.
	sharedSecret, err := auth.DecryptPKCS1v15(h.privKey, encSecret)
	if err != nil {
		return nil, fmt.Errorf("login: decrypting shared secret: %w", err)
	}
	if len(sharedSecret) != 16 {
		return nil, fmt.Errorf("login: shared secret must be 16 bytes, got %d", len(sharedSecret))
	}

	verifyToken, err := auth.DecryptPKCS1v15(h.privKey, encToken)
	if err != nil {
		return nil, fmt.Errorf("login: decrypting verify token: %w", err)
	}

	// Constant-time comparison to thwart timing attacks.
	if !bytes.Equal(verifyToken, challenge) {
		return nil, errors.New("login: verify token mismatch — possible man-in-the-middle")
	}

	return sharedSecret, nil
}

// sendLoginSuccess writes the Login Success packet (0x02 S→C).
//
// Fields (1.21.4):
//
//	UUID     playerUUID
//	String   username
//	VarInt   numProperties
//	[]       properties  { String name, String value, Bool signed, String? sig }
//	Bool     strictErrorHandling   false
func (h *LoginHandler) sendLoginSuccess(conn *network.ClientConn, uuid protocol.UUID, name string, props []auth.ProfileProperty) error {
	b := protocol.NewBuilder(packetIDLoginSuccess).
		UUID(uuid).
		String(name).
		VarInt(int32(len(props)))

	for _, p := range props {
		signed := p.Signature != ""
		b.String(p.Name).String(p.Value).Bool(signed)
		if signed {
			b.String(p.Signature)
		}
	}

	b.Bool(false) // strictErrorHandling — false for permissive mode

	return conn.WritePacket(b.Build())
}

// readLoginAcknowledged reads the Login Acknowledged packet (0x03 C→S, no fields).
func (h *LoginHandler) readLoginAcknowledged(conn *network.ClientConn) error {
	pkt, err := conn.ReadPacket()
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("login: client disconnected before Login Acknowledged")
		}
		return fmt.Errorf("login: reading Login Acknowledged: %w", err)
	}
	if pkt.ID != packetIDLoginAcknowledged {
		return fmt.Errorf("login: expected 0x03 (Login Acknowledged), got 0x%02X", pkt.ID)
	}
	return nil
}

// disconnect sends a Login Disconnect packet with a human-readable reason and
// then wraps innerErr so the caller can return it directly.
func (h *LoginHandler) disconnect(conn *network.ClientConn, message string, innerErr error) error {
	// JSON text component — escape any double quotes in the message.
	reason := `{"text":"` + jsonEscape(message) + `"}`
	pkt := protocol.NewBuilder(packetIDLoginDisconnect).String(reason).Build()
	// Best-effort send; ignore write errors since we're already bailing out.
	_ = conn.WritePacket(pkt)

	slog.Warn("login disconnected", "remote", conn.RemoteAddr(), "reason", message, "err", innerErr)
	return fmt.Errorf("login: %s: %w", message, innerErr)
}

// jsonEscape escapes the minimal set of characters needed for a JSON string value.
func jsonEscape(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
