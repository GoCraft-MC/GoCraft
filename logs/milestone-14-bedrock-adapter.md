# GoCraft — Milestone 14: Bedrock Adapter (Beta)

**Date:** 2026-08-17
**Protocol:** Minecraft Bedrock Edition 1.26.40 (RakNet / MCPE protocol 2168)

---

## What was completed

### Bedrock listener
- `bedrock/listener.go` wraps the `go-raknet` + `gophertunnel` stack
  - `NewListener(addr, config)` — binds UDP port (default 19132)
  - `Accept()` → per-client goroutine for Bedrock session handling
  - `BroadcastMessage(text string)` — sends §-coded text to all Bedrock clients
  - `SetCustomItemEntries(entries []protocol.ItemEntry)` — injects custom items into StartGame
  - `SetResourcePacks(packs []*resource.Pack)` — bulk resource pack registration

### StartGame handshake
- Sends `StartGame` packet with:
  - World name, seed, dimension, game mode, difficulty
  - Spawn position matching Java world's spawn
  - Item registry: vanilla items from `bedrockItemRegistry()` + custom item entries
  - Runtime entity ID for the joining player
  - `texturepacksRequired: true` when resource packs are configured

### Resource pack delivery
- `.mcpack`, `.zip` loaded via `resource.ReadPath`
- `.mcaddon` extracted: finds subdirectories containing `manifest.json`, extracts each to a temp directory, loads with `resource.ReadPath`
- `loadBedrockPackFromBytes(data []byte)` — writes to temp file for in-memory pack injection (custom items mcaddon)
- Client receives `ResourcePacksInfo` + `ResourcePackStack` before login completes

### Cross-play bridge
- `server/server.go` registers a message observer on the Java session manager
- Non-chat system messages (join, quit, death) observed → forwarded to `bedrockListener.BroadcastMessage`
- Chat messages: Java sessions receive MiniMessage-formatted string; Bedrock clients receive `applyBedrock` output (gradients collapsed, hex mapped to nearest named color)
- Player position changes in `core/player.Player` visible to both adapters — shared canonical state

### Bedrock login flow
- Reads `Login` packet, validates Xbox Live JWT chain (online mode) or accepts guest UUID (offline mode)
- Creates `core/player.Player` with Bedrock flag set; registers with game core
- Sends: `PlayStatus` → `ResourcePacksInfo` → `ResourcePackStack` → `StartGame` → `BiomeDefinitionList` → `CreativeContent` → `PlayStatus(LoginSuccess)`

---

## Design principles

The Bedrock adapter reads from `core/` independently. It does **not** consume or translate Java packets. Both adapters write to and read from the same canonical `core/player.Player`, `core/world.World`, and `core/entity.Registry`. This is not a proxy like Geyser — it is a native dual-protocol server.

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Bedrock clients can join | ✅ M14 |
| Resource pack delivery (mcpack, mcaddon, zip) | ✅ M14 |
| Cross-play chat (Bedrock-safe formatting) | ✅ M14 |
| Join/quit announcements forwarded | ✅ M14 |
| Custom item entries in StartGame | ✅ M14 |
| Bedrock movement → world update | Beta |
| Full parity with Java feature set | Future (M15+) |

---

## Architecture additions

```
bedrock/
├── listener.go        ← UDP listener, StartGame, resource packs, BroadcastMessage
└── doc.go             ← package doc

server/
├── resourcepack.go    ← loadMcaddon, loadBedrockPackFromBytes
└── server.go          ← cross-play bridge, observer wiring
```
