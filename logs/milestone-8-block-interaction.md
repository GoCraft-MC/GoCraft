# GoCraft — Milestone 8: Block Interaction

**Date:** 2026-08-04
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Block place & break
- Handles `Player Action` (0x23) — start/finish dig, drop item, swap hands
- Handles `Use Item On` (0x36) — block placement with face + cursor position
- Server validates placement against held item and target block face
- Block state updated in `core/world.Chunk` and broadcast as `Block Update` (0x09) to all sessions
- Block break animation: `Block Destroy Stage` (0x06) packets sent at 10% intervals during dig
- Instant-break materials (e.g. grass, flowers) skip animation

### Interaction events
- `Interact` (0x16) packet handled: attack / interact / interact_at
- Entity hit: applies basic damage to target player (fall damage, combat M10+)
- `Player Block Placement` validates placement isn't inside the player hitbox

### Block entity sync
- Block entities (chests, signs, etc.) serialised as NBT in the Chunk Data packet
- On block update, affected block entity data is re-sent via `Block Entity Data` (0x0B)

### Sound effects
- `Sound Effect` (0x65) sent on block break/place with hardcoded pitch/volume
- Sound IDs loaded from `data/sounds.json` (data-driven registry, M13)

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Block break (with animation) | ✅ M8 |
| Block placement | ✅ M8 |
| Block update broadcast to all players | ✅ M8 |
| Basic entity interaction (hit) | ✅ M8 |
| Block entity data sync | ✅ M8 |
| World persistence (blocks survive restart) | ❌ M9 |
| Inventory / item management | ❌ M10 |

---

## Architecture additions

```
java/handler/
├── blockinteract.go   ← Player Action, Use Item On, block break/place logic
└── interact.go        ← Interact packet, entity hit dispatch

core/world/
└── chunk.go           ← SetBlock, GetBlock, block entity map
```
