package auth

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"
)

const sessionServerURL = "https://sessionserver.mojang.com/session/minecraft/hasJoined"

// GameProfile is the player identity returned by the Mojang session server.
type GameProfile struct {
	// ID is the player UUID as a hex string without dashes (e.g. "4566e69fc90748ee8d71d7ba5aa00d20").
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Properties []ProfileProperty `json:"properties"`
}

// ProfileProperty holds a single signed property, typically "textures".
type ProfileProperty struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	Signature string `json:"signature,omitempty"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// HasJoined calls the Mojang session server to verify that a player has joined.
//
// username is the name from the Login Start packet.
// serverIDHash is the signed hex SHA1 digest (see ComputeServerHash).
// remoteAddr is the client's TCP address; the IP is forwarded to Mojang so
// they can detect proxy abuse (vanilla behaviour sends the IP unconditionally).
func HasJoined(username, serverIDHash string, remoteAddr net.Addr) (*GameProfile, error) {
	url := fmt.Sprintf("%s?username=%s&serverId=%s", sessionServerURL, username, serverIDHash)

	// Forward the client IP for proxy detection.
	if host, _, err := net.SplitHostPort(remoteAddr.String()); err == nil && host != "" {
		url += "&ip=" + host
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("session: hasJoined request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusForbidden:
		return nil, fmt.Errorf("session: authentication rejected (status %d) — player may not be logged in", resp.StatusCode)
	case http.StatusOK:
		// proceed
	default:
		return nil, fmt.Errorf("session: unexpected status %d from Mojang", resp.StatusCode)
	}

	var profile GameProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("session: decoding profile: %w", err)
	}
	if profile.ID == "" || profile.Name == "" {
		return nil, fmt.Errorf("session: empty profile returned by Mojang")
	}
	return &profile, nil
}

// ComputeServerHash computes the "server ID hash" sent to the Mojang session API.
//
// Minecraft defines it as:
//
//	SHA1( serverID + sharedSecret + serverPublicKey )
//
// where serverID is always the empty byte slice in modern Minecraft.
// The result is formatted as a signed hex integer (see minecraftHexDigest).
func ComputeServerHash(serverID, sharedSecret, publicKeyDER []byte) string {
	h := sha1.New()
	h.Write(serverID)
	h.Write(sharedSecret)
	h.Write(publicKeyDER)
	return minecraftHexDigest(h.Sum(nil))
}

// minecraftHexDigest converts a raw 20-byte SHA1 hash to the Minecraft
// "hex digest" format: a lowercase hexadecimal string representing the hash
// as a signed two's-complement big integer (negative values have a leading "-").
func minecraftHexDigest(hash []byte) string {
	negative := hash[0]&0x80 != 0
	if negative {
		// Two's complement negation: flip all bits, then add 1.
		for i := range hash {
			hash[i] = ^hash[i]
		}
		for i := len(hash) - 1; i >= 0; i-- {
			hash[i]++
			if hash[i] != 0 {
				break // no carry out
			}
			// hash[i] wrapped back to 0 → carry propagates
		}
	}

	hex := new(big.Int).SetBytes(hash).Text(16)
	hex = strings.TrimLeft(hex, "0")
	if hex == "" {
		hex = "0"
	}
	if negative {
		hex = "-" + hex
	}
	return hex
}
