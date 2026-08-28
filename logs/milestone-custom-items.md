# GoCraft — Custom Items System (In Progress)

**Branch:** `feature/custom-items`
**Date:** 2026-08-28
**Protocol:** Java 1.21.4 (769) · Bedrock 1.26.40 (2168)

---

## Goal

Define custom items once in a YAML pack, and have GoCraft auto-generate the correct delivery format for each edition at startup — no manual resource pack editing required.

---

## What was completed

### Pack format
- `packs/<pack-name>/items.yml` — item definitions:
  ```yaml
  namespace: myplugin
  items:
    - id: magic_sword
      display_name: "<gradient:#FF0000:#FF8800>Magic Sword</gradient>"
      material: diamond_sword
      texture: textures/magic_sword.png
      lore:
        - "<gray>Deals extra damage"
  ```
- Items use `namespace:id` format (e.g. `myplugin:magic_sword`)
- Textures referenced relative to the pack directory

### Persistent registry
- `packs/.registry.yml` — maps each `namespace:id` to a stable CMD + Bedrock runtime ID
- CMD range: 30100+ (matches CustomiZer convention, avoids vanilla conflicts)
- Bedrock runtime ID range: 5000+ (above all vanilla IDs ~750)
- IDs assigned on first seen and never reassigned — safe to add new items without breaking existing ones
- New IDs appended; removed items leave gaps (IDs not reused)

### Java resource pack generation
- `customitems/javapack.go`: `BuildJavaPack()` generates a ZIP containing:
  - `pack.mcmeta`
  - Per-material override JSON: adds `{predicate: {custom_model_data: CMD}, model: "namespace/id"}` entry
  - Per-item model JSON: `{parent: "item/handheld", textures: {layer0: "namespace/id"}}`
  - Texture PNGs from pack directory
- Pack served over HTTP (`customitems/javaserver.go`) — SHA-1 hash sent in resource pack push

### Bedrock pack generation
- `customitems/bedrockpack.go`: `BuildBedrockPack()` generates an `.mcaddon` containing:
  - `gocraft_rp/` — resource pack with `item_texture.json` + PNG textures
  - `gocraft_bp/` — behavior pack with per-item component JSON (`minecraft:icon`, `minecraft:display_name`, `minecraft:max_stack_size`)
- Fixed UUIDs for both packs (deterministic — client caches correctly between restarts)
- Loaded by `server/resourcepack.go` via `loadBedrockPackFromBytes`

### Bedrock StartGame injection
- `customitems/bedrockitems.go`: `BedrockItemEntries()` returns `[]protocol.ItemEntry` with `ComponentBased: true`
- Injected into `StartGame.Items` alongside vanilla registry

### Config
- `server.yml` custom items section:
  ```yaml
  custom_items:
    enabled: true
    packs_dir: packs
    java:
      serve_port: 8080
      public_host: ""   # auto-detected from server bind address
  ```

---

## Known limitations

| Limitation | Notes |
|-----------|-------|
| Bedrock behavior pack items are cosmetic only | GoCraft is not a vanilla Bedrock engine; behavior pack components (attack damage, durability) are not executed |
| Java CMD technique requires the base material to exist in the player's hand | Item renders correctly; server-side logic must be implemented separately |
| No crafting recipes yet | Items can only be given via `/give` |
| Texture format | Must be 16×16 PNG; higher resolution works on Java, Bedrock requires power-of-two |

---

## Architecture

```
customitems/
├── pack.go          ← ItemDef, PackDef, LoadedPack, ResolvedItem
├── registry.go      ← persistent .registry.yml, ID assignment
├── manager.go       ← Load(packsDir), Items(), IsEmpty()
├── javapack.go      ← BuildJavaPack() → ZIP bytes + SHA-1
├── javaserver.go    ← HTTP server for Java pack delivery
├── bedrockpack.go   ← BuildBedrockPack() → .mcaddon bytes
└── bedrockitems.go  ← BedrockItemEntries() → []protocol.ItemEntry

docs/
└── custom-items.md  ← full admin guide
```

---

## Status

- [x] Pack loader + persistent registry
- [x] Java resource pack generation + HTTP delivery
- [x] Bedrock .mcaddon generation + injection
- [x] Bedrock StartGame item entry injection
- [x] Config section in server.yml
- [x] Admin documentation
- [ ] Server-side item behavior hooks (future)
- [ ] Crafting recipe support (future)
