# GoCraft — Milestone 4: World I/O & Chunk Streaming

**Date:** 2026-07-30
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### World I/O
- Implemented Anvil region file reader (`core/world/region.go`)
  - Parses `.mca` files from `world/region/` (Java Anvil format)
  - Handles sector offsets, chunk timestamps, zlib/gzip/uncompressed NBT
  - Returns raw chunk NBT compound for further parsing
- Flat-world generator as fallback when no region files exist
  - Generates bedrock + stone layers + surface grass in a 64×64 area
  - Stores generated chunks in an in-memory map; no persistence until M9
- `core/world/world.go`: `World` type with `GetChunk(cx, cz int32)` — reads from disk, falls back to generator

### Chunk packet serialization
- `java/handler/chunk.go`: converts loaded `core/world/Chunk` to a Chunk Data And Update Light packet (0x27)
  - Writes section bitmask, per-section block state palette + data array
  - Computes per-section sky light and block light arrays (all-max for now)
  - Writes heightmap NBT (MOTION_BLOCKING, WORLD_SURFACE)
  - Sends 0 block entities
- Chunk radius: configurable via `view_distance` in `server.yml` (default 8)

### Play-loop integration
- On player spawn, server sends all chunks in the view circle before the player can move
- `Set Center Chunk` sent on first login and on chunk border crossing
- Chunks are sent in spiral order (center-out) to minimize visible pop-in

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Handshake + ping | ✅ M1 |
| Login + auth | ✅ M2 |
| Configuration state | ✅ M3 |
| Play state entry | ✅ M3 |
| World file loading (Anvil .mca) | ✅ M4 |
| Flat-world fallback generator | ✅ M4 |
| Chunk streaming to client | ✅ M4 |
| Player movement | ❌ M5 |
| Chat | ❌ M7 |
| Bedrock support | ❌ M14 |

---

## Architecture additions

```
core/
└── world/
    ├── world.go      ← GetChunk, in-memory cache
    ├── region.go     ← Anvil .mca reader
    ├── chunk.go      ← Chunk type (sections, biomes)
    └── generator.go  ← flat-world fallback
java/
└── handler/
    └── chunk.go      ← Chunk Data And Update Light packet builder
```

---

## Known limitations at this milestone

- Biome data is placeholder (plains everywhere)
- Block entities not serialised yet
- No chunk unloading — all loaded chunks stay in memory
- Sky/block light values are hardcoded maximums (no real lighting engine)
