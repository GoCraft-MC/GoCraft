# 🔐 GoCraft — Milestone 2 Complete: Login & Authentication

**Project:** GoCraft — Minecraft Java Edition server written entirely in Go
**Date:** 2026-07-29
**Milestone:** #2 — Login & Authentication

---

## ✅ What was accomplished

A real Minecraft Java Edition client can now fully log in to GoCraft. The complete authentication handshake — including RSA key exchange, Mojang session verification, and AES encryption — is implemented from scratch in Go with no JVM.

### What works right now

**Online mode (online_mode: true in server.yml):**
- Generates an RSA-1024 keypair at server startup
- Sends Encryption Request with the public key and a random 4-byte challenge
- Receives Encryption Response (client sends the shared secret + challenge encrypted with our public key)
- Decrypts both with PKCS#1 v1.5 and verifies the challenge matches
- Computes the server ID hash: `SHA1("" + sharedSecret + publicKeyDER)` as a signed hex digest
- Calls `sessionserver.mojang.com/session/minecraft/hasJoined` to verify the player
- Enables AES-128-CFB8 encryption on the connection (key = IV = shared secret)
- Sends Login Success (encrypted) with the Mojang-verified UUID and texture properties
- Receives Login Acknowledged

**Offline mode (online_mode: false in server.yml):**
- Generates a deterministic UUID v3 from `"OfflinePlayer:" + name` (matches vanilla behaviour)
- Sends Login Success directly (no encryption)
- Receives Login Acknowledged

**Both modes:**
- Sends a friendly Configuration Disconnect (`"Welcome, <name>! Milestone 3 not yet implemented"`) so the client shows a readable message instead of a generic error
- Full structured logging: player name, UUID, mode, remote address

---

## 📦 New code in this milestone

### New packages

| Package | File | Purpose |
|---------|------|---------|
| `auth` | `keypair.go` | RSA-1024 keygen, DER public key marshalling, PKCS#1 v1.5 decrypt |
| `auth` | `cfb8.go` | AES-128-CFB8 stream cipher (Go stdlib only has CFB-128, so we implement CFB8) |
| `auth` | `session.go` | Mojang `hasJoined` API call + signed SHA1 hex digest |
| `auth` | `uuid.go` | Offline UUID v3 + Mojang UUID string parsing |
| `protocol` | `uuid.go` | `UUID [16]byte` type with `ReadUUID` / `WriteUUID` |

### Modified files

| File | What changed |
|------|--------------|
| `protocol/types.go` | Added `ReadByteArray`, `WriteByteArray` (VarInt-prefixed byte slices) |
| `protocol/packet.go` | Added `ByteArray()` and `UUID()` methods to the packet Builder |
| `network/conn.go` | Added `writer io.Writer` field + `EnableEncryption(sharedSecret []byte)` |
| `handler/login.go` | **New** — full login state handler (all 7 steps) |
| `server/server.go` | Generates RSA keypair at startup, routes login connections |
| `main.go` | Handles `server.New()` now returning an error |

---

## 🧪 Tests

**28 tests, all passing.** New tests in this milestone:

| Test | What it covers |
|------|---------------|
| `TestCFB8_RoundTrip` | Encrypt then decrypt gives back original plaintext |
| `TestCFB8_Incremental` | Byte-by-byte cipher == bulk cipher (stateful correctness) |
| `TestCFB8_KnownVector` | Cipher produces non-trivial output |
| `TestMinecraftHexDigest_KnownPositive` | SHA1("Notch") → correct positive hex |
| `TestMinecraftHexDigest_KnownNegative` | SHA1("jeb_") → correct negative hex with leading "-" |
| `TestMinecraftHexDigest_Zero` | All-zero hash → "0" |
| `TestOfflineUUID_KnownValue` | Version 3 and RFC 4122 variant bits are set correctly |
| `TestOfflineUUID_Deterministic` | Same name always gives same UUID |
| `TestOfflineUUID_Unique` | Different names give different UUIDs |
| `TestParseMojangUUID_WithoutDashes` | Parses 32-char hex UUID string |
| `TestParseMojangUUID_WithDashes` | Parses standard dashed UUID string |
| `TestParseMojangUUID_Invalid` | Bad input returns an error |
| `TestGenerateKeyPair` | Key is 1024 bits and DER-marshals cleanly |
| `TestDecryptPKCS1v15_RoundTrip` | RSA key survives DER round-trip |
| `TestProtocolUUID_RoundTrip` | UUID writes 16 bytes and reads back identical |

---

## 🌍 Cross-platform builds confirmed

| Platform | Status |
|----------|--------|
| Windows (amd64) | ✅ |
| Linux (amd64) | ✅ |
| macOS (amd64) | ✅ |

---

## 🗺️ Roadmap

| # | Milestone | Status |
|---|-----------|--------|
| 1 | Handshake + Server List Ping | ✅ Done |
| 2 | Login & Authentication | ✅ Done |
| 3 | Configuration state (registry data, feature flags) | 🔜 Next |
| 4 | World I/O (region files, NBT, chunk loading/saving) | ⬜ |
| 5 | Chunk generation (flat + basic noise) | ⬜ |
| 6 | Player joins (spawn, chunk sending, player list) | ⬜ |
| 7 | Movement & chat | ⬜ |
| 8 | Blocks & inventories | ⬜ |
| 9 | Entities & physics | ⬜ |
| 10 | Scheduler & event system | ⬜ |
| 11 | Go-native plugin API | ⬜ |
| 12 | Commands & permissions | ⬜ |

---

*GoCraft is a clean-room Go implementation of the Minecraft Java Edition server protocol.
No Mojang, CraftBukkit, or Paper source code was copied.*
