# GoCraft — Milestone 10: Inventory & Items

**Date:** 2026-08-07
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Inventory model
- `core/player/inventory.go`: 46-slot Java inventory (hotbar 0–8, main 9–35, armor 36–39, offhand 45)
- `ItemStack` type: namespace ID, count, damage value, NBT tag map
- `Inventory.Set`, `Get`, `Swap`, `AddItem`, `RemoveItem` — thread-safe via sync.Mutex

### Window / container packets
- `Set Container Content` (0x13) sent on join to initialize client inventory
- `Set Container Slot` (0x15) sent on single-slot changes (pick-up, place, crafting result)
- `Set Held Item` (0x53) sent when server changes active hotbar slot
- Handles `Set Held Item` (C→S, 0x2F) — updates `Player.HeldSlot`
- Handles `Click Container` (0x11) — inventory click with slot, button, action type
  - Action types: normal click, shift-click, hotbar swap, drop, double-click
  - Server validates and applies the action; sends corrective `Set Container Content` on mismatch

### Item NBT encoding
- Items encoded as Network NBT in slot data (1.20.5+ "item component" format)
- Core components: `minecraft:item_name`, `minecraft:lore`, `minecraft:custom_model_data`
- `nbtLoreTextComponent` used for styled lore lines (italic override to match vanilla)

### Drop / pickup
- `Player Action` DROP_ITEM and DROP_ITEM_STACK spawn a dropped item entity
- Item entities swept up when a player walks within 1 block (tick-based check, M11)

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Inventory display on join | ✅ M10 |
| Click-to-move items | ✅ M10 |
| Hotbar slot switching | ✅ M10 |
| Item drop | ✅ M10 |
| Item NBT components | ✅ M10 |
| Entity system (item entities, pickup) | ❌ M11 |
| Commands | ❌ M12 |

---

## Architecture additions

```
core/player/
└── inventory.go    ← ItemStack, Inventory, 46-slot model

java/handler/
└── inventory.go    ← Click Container, Set Held Item, slot packet builders
```
