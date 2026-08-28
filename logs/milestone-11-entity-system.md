# GoCraft — Milestone 11: Entity System

**Date:** 2026-08-09
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Core entity model
- `core/entity/entity.go`: base `Entity` struct — ID, UUID, type ID, position, velocity, on-ground flag
- `core/entity/registry.go`: `Registry` — thread-safe map of entity ID → Entity; atomic ID counter shared with M6
- Entity types: `EntityTypeItem`, `EntityTypePlayer`, plus extensible numeric IDs loaded from `data/entities.json`

### World entity tracking
- `core/world/world.go` gains an entity registry per dimension
- `AddEntity`, `RemoveEntity`, `EntitiesNear(x, y, z, radius)` — used for item pickup sweep and mob targeting

### Entity spawn / despawn packets
- `Add Entity` (0x01) sent to players when a new non-player entity enters their view
- `Add Player` (0x3C) for player entities (unchanged from M6, now managed via entity registry)
- `Remove Entities` (0x42) sent when entity leaves view or is removed

### Entity metadata
- `Set Entity Metadata` (0x57) sent after spawn for type-specific fields (item stack, health, flags)
- Metadata format: typed entries encoded as VarInt tag + type ID + value

### Item entities
- Dropped items spawn as `EntityTypeItem` with a `minecraft:item` metadata entry
- Server ticks item entities every 20 ticks: check for players within 1 block, add to inventory, remove entity
- Items despawn after 300 seconds (6000 ticks) if not picked up

### Entity motion
- `Set Entity Velocity` (0x5D) sent when server applies impulse (e.g. knockback, item toss)
- Gravity simulation: item entities fall 0.04 blocks per tick until on-ground

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Entity registry | ✅ M11 |
| Item drop + pickup | ✅ M11 |
| Item entity despawn timer | ✅ M11 |
| Entity metadata sync | ✅ M11 |
| Player entity managed via registry | ✅ M11 |
| Commands | ❌ M12 |
| Data-driven registries | ❌ M13 |

---

## Architecture additions

```
core/entity/
├── entity.go     ← base Entity type, velocity, flags
└── registry.go   ← thread-safe entity map, ID counter

core/world/
└── world.go      ← entity registry integration, EntitiesNear

java/handler/
└── entity.go     ← spawn/despawn packets, metadata builder, tick handler
```
