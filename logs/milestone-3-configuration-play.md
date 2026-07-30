# GoCraft — Milestone 3: Configuration + Play State Entry

**Date:** 2026-07-29
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Architectural refactoring (pre-M3)
- Migrated all Java-specific code into `java/` subdirectories:
  - `java/protocol/` — VarInt, packet framing, wire types, UUID
  - `java/auth/` — CFB8 cipher, RSA keypair, Mojang session, offline UUID
  - `java/network/` — TCP listener, ClientConn with encryption
  - `java/handler/` — Handshake, Status, Login handlers
- Created edition-agnostic core:
  - `core/spatial/` — Vec3, BlockPos, Rotation
  - `core/player/` — Player model (no Java/Bedrock specifics)
  - `core/game/` — Game core with sync.Map player registry + atomic entity IDs
- Created `bedrock/doc.go` stub (Milestone 14 placeholder)
- Deleted old `auth/`, `protocol/`, `network/`, `handler/` directories
- Updated `server/server.go` to use `java/` packages and the game core

### Milestone 3: Configuration state
**`java/handler/config.go`**
- Known Packs optimization: server declares `minecraft:core` v1.21.4
- Client confirms it has the pack cached → no Registry Data needed
- Plugin Message: `minecraft:brand` = "GoCraft"
- Feature Flags: `minecraft:vanilla`
- Update Tags: 0 registries (client uses cached data)
- Finish Configuration → waits for Acknowledge Finish

### Milestone 3: Play state entry
**`java/handler/play.go`**
- Login (Play) packet (0x2C) — entity ID, dimension, gamemode, view distance
- Player Abilities (0x3A) — survival flags, flight speed
- Set Default Spawn Position (0x5B) — packed 64-bit block position
- Player Info Update (0x40) — adds player to tab list (ADD_PLAYER + UPDATE_LISTED)
- Synchronize Player Position (0x42) — with velocity fields (1.21+) and teleport ID
- Set Center Chunk (0x58) — chunk streaming anchor
- Game Event reason=13 (0x23) — "start waiting for level chunks" (critical for 1.21.4)
- Keep-Alive loop — sends every 10s, 30s timeout

### Protocol extensions added
- `java/protocol/types.go`: `ReadFloat`, `WriteFloat`, `ReadDouble`, `WriteDouble`
- `java/protocol/packet.go` Builder: `.Int()`, `.Byte()`, `.Float()`, `.Double()`

### Server wiring
- `server/server.go` now routes Login → Configuration → Play
- `registerPlayer()` creates `core/player.Player`, registers with game core, updates online count

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Server-list ping (MOTD, player count) | ✅ M1 |
| Online-mode authentication (RSA + Mojang) | ✅ M2 |
| Offline-mode (UUID v3) | ✅ M2 |
| AES-128-CFB8 encryption | ✅ M2 |
| Configuration state (Known Packs, brand, feature flags) | ✅ M3 |
| Play state entry (Login, Abilities, Position, Keep-Alive) | ✅ M3 |
| Player tab list | ✅ M3 |
| Chunk sending / world | ❌ M4+ |
| Chat | ❌ M5+ |
| Player movement | ❌ M4+ |
| Bedrock support | ❌ M14 |

---

## Architecture

```
GoCraft/
├── main.go
├── server/server.go          ← orchestrator (Login→Config→Play)
├── config/config.go          ← shared YAML config
├── core/
│   ├── game/game.go          ← edition-agnostic player registry
│   ├── player/player.go      ← canonical Player model
│   └── spatial/spatial.go    ← Vec3, BlockPos, Rotation
├── java/
│   ├── auth/                 ← RSA, CFB8, Mojang session, UUID
│   ├── handler/              ← Handshake, Status, Login, Config, Play
│   ├── network/              ← TCP listener, ClientConn, encryption
│   └── protocol/             ← VarInt, packet framing, types, UUID
└── bedrock/doc.go            ← M14 placeholder
```

---

## Test results

```
ok  GoCraft/java/auth     (15 tests: CFB8, SHA1 digest, UUID, RSA)
ok  GoCraft/java/protocol (VarInt encoding/decoding, packet framing)
```

All tests passing. Build clean on Windows (GOOS=windows GOARCH=amd64).
