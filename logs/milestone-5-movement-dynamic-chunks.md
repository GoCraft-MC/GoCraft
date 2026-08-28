# GoCraft — Milestone 5: Movement & Dynamic Chunk Loading

**Date:** 2026-07-31
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Player movement
- Handles all four C→S movement packets:
  - `Confirm Teleport` (0x00) — validates pending teleport IDs
  - `Set Player Position` (0x1A)
  - `Set Player Position And Rotation` (0x1B)
  - `Set Player Rotation` (0x1C)
  - `Set Player On Ground` (0x1D)
- Server stores authoritative position in `core/player.Player` (`X`, `Y`, `Z`, `Yaw`, `Pitch`)
- Basic anti-cheat: position delta capped at 100 blocks per tick; out-of-range updates are silently ignored and player is teleported back

### Dynamic chunk loading / unloading
- On each position update, server computes the player's current chunk (`cx`, `cz`)
- Compares current chunk set against previously sent chunk set
- Newly visible chunks are sent as Chunk Data packets
- Chunks that left the view distance are sent a Forget Level Chunk (0x1F) packet
- `Set Center Chunk` is resent on every chunk boundary crossing
- View distance controlled by `view_distance` in `server.yml`

### Chunk cache
- `core/world/World` now has a two-level cache:
  - In-memory `map[chunkKey]*Chunk` — loaded/generated chunks
  - Dirty set for chunks modified since last save (M9 will flush these)

### Player on-ground tracking
- `Player.OnGround bool` updated per packet
- Used later by damage and fall logic (M8+)

---

## Current capabilities

| Feature | Status |
|---------|--------|
| World loading + chunk streaming | ✅ M4 |
| Player movement (all 4 packets) | ✅ M5 |
| Dynamic chunk load/unload | ✅ M5 |
| Anti-cheat position clamping | ✅ M5 |
| Multiplayer sync (other players see movement) | ❌ M6 |
| Chat | ❌ M7 |

---

## Architecture additions

```
java/handler/
└── movement.go   ← Confirm Teleport, Position, Rotation, OnGround handlers

core/world/
└── world.go      ← dirty set, chunk unload tracking
```
