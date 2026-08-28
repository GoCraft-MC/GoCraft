# GoCraft — Milestone 13: Data-Driven Registries

**Date:** 2026-08-13
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Registry loader
- `data/` directory contains versioned JSON files checked into the repo
- `core/registries/` package loads them at startup via `Load(dataDir)`:
  - `blocks.json` — block name → default state ID, all state combinations
  - `items.json` — item name → numeric ID (used for slot encoding)
  - `entities.json` — entity type name → numeric type ID
  - `biomes.json` — biome name → numeric ID (used in chunk sections)
  - `packet_ids.json` — packet name → ID for each protocol version (currently only 769)
- All maps stored as module-level singletons, read-only after startup

### Block state IDs
- Block placement uses `blocks.json` to resolve the correct state ID before writing to chunk
- Block updates sent with accurate state IDs (not hardcoded defaults)
- `GetBlockState(name string, props map[string]string) int` — resolves the exact state

### Packet ID map
- Packet IDs for outgoing packets looked up from `packet_ids.json` instead of hardcoded constants
- Allows protocol version upgrades by editing JSON — no Go recompile for ID-only changes

### Item ID registry
- `/give` command resolves item names to numeric IDs via `items.json`
- Inventory slot encoding uses registry IDs for correct client rendering

### Biome registry
- Chunk sections encode biome IDs from `biomes.json`
- Previously all sections used placeholder `plains` (ID 1); now accurate to world data

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Block state IDs from JSON | ✅ M13 |
| Item IDs from JSON | ✅ M13 |
| Entity type IDs from JSON | ✅ M13 |
| Biome IDs from JSON | ✅ M13 |
| Packet IDs from JSON | ✅ M13 |
| Bedrock adapter | ❌ M14 |

---

## Architecture additions

```
data/
├── blocks.json
├── items.json
├── entities.json
├── biomes.json
└── packet_ids.json

core/registries/
├── registries.go   ← Load, singletons
├── blocks.go       ← block state resolver
└── items.go        ← item ID lookup
```
