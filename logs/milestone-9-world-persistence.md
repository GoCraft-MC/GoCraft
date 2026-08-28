# GoCraft — Milestone 9: World Persistence

**Date:** 2026-08-05
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Anvil region file writer
- `core/world/region.go` extended with write path
  - Allocates sectors sequentially; rewrites the 8 KiB header on each flush
  - Chunks serialised to compressed NBT (zlib, compression type 2)
  - Atomic write: data written to `.mca.tmp` then renamed to avoid corruption on crash

### Autosave scheduler
- `core/world/world.go`: background goroutine flushes dirty chunks every N seconds (configurable via `autosave_interval` in `server.yml`, default 60s)
- Dirty set cleared after successful write
- Graceful shutdown (`SIGINT`/`SIGTERM`) flushes all dirty chunks before exit

### Level data
- `level.dat` read on startup to load spawn point, world name, game rules
- `level.dat` written on clean shutdown with updated `LastPlayed` and player count

### Memory-mode option
- `world.memory_mode: true` in `server.yml` keeps all chunks in RAM; disables all disk writes
- Intended for ephemeral / test servers that reset on restart

### Player data persistence
- `playerdata/<uuid>.dat` (NBT) saved on disconnect and every autosave cycle
- Stores: position, health, food, inventory, game mode
- Loaded on join if file exists; new players get default spawn + full health

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Block changes survive server restart | ✅ M9 |
| Autosave with dirty tracking | ✅ M9 |
| Atomic region file writes | ✅ M9 |
| Player data persistence | ✅ M9 |
| Memory-mode (no disk I/O) | ✅ M9 |
| Inventory UI | ❌ M10 |
| Entity persistence | ❌ M11 |

---

## Architecture additions

```
core/world/
├── world.go     ← autosave loop, dirty set flush, shutdown hook
├── region.go    ← Anvil reader + writer, atomic rename
└── level.go     ← level.dat read/write
```
