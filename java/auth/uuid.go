package auth

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"GoCraft/java/protocol"
)

// OfflineUUID generates the deterministic offline-mode UUID for a player name.
//
// This matches the vanilla Minecraft algorithm:
//
//	UUID.nameUUIDFromBytes(("OfflinePlayer:" + name).getBytes(UTF_8))
//
// which is UUID version 3 (MD5-based namespace UUID).
func OfflineUUID(name string) protocol.UUID {
	h := md5.Sum([]byte("OfflinePlayer:" + name))
	// Set UUID version to 3 (MD5).
	h[6] = (h[6] & 0x0F) | 0x30
	// Set UUID variant to RFC 4122.
	h[8] = (h[8] & 0x3F) | 0x80
	return protocol.UUID(h)
}

// ParseMojangUUID parses the UUID string returned by the Mojang session API.
// Mojang returns UUIDs without dashes (e.g. "4566e69fc90748ee8d71d7ba5aa00d20").
func ParseMojangUUID(s string) (protocol.UUID, error) {
	// Accept both with-dashes and without-dashes forms.
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return protocol.UUID{}, fmt.Errorf("auth: UUID string has wrong length %d (want 32 hex chars)", len(s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return protocol.UUID{}, fmt.Errorf("auth: parsing UUID %q: %w", s, err)
	}
	var u protocol.UUID
	copy(u[:], b)
	return u, nil
}
