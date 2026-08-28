# Custom Items

GoCraft has a built-in cross-edition custom item system. You define items once in YAML, drop textures in a folder, and GoCraft automatically:

- Generates a **Java resource pack** (Custom Model Data overrides + textures) and serves it over HTTP so Java clients see the correct visuals.
- Generates a **Bedrock `.mcaddon`** (resource pack + behavior pack) and pushes it to Bedrock clients on login.
- Registers custom item IDs in the Bedrock **StartGame** packet so Bedrock clients recognise the items natively.

No separate pack-building step, no Geyser required.

---

## Folder layout

```
packs/
  mypacks/
    items.yml          ← item definitions
    textures/
      ruby.png         ← 16×16 PNG textures
      ruby_sword.png
  .registry.yml        ← auto-generated; do not edit
```

Each sub-directory under `packs/` is one pack. The `packs/` location is configurable via `custom_items.packs_dir` in `server.yml`.

---

## items.yml format

```yaml
namespace: mypacks   # unique lowercase identifier, no spaces

items:
  ruby:
    display_name: "<red>Ruby"
    material: paper          # vanilla Java base item (lowercase)
    texture: ruby.png        # filename inside textures/

  ruby_sword:
    display_name: "<gradient:#ff0000:#aa0000>Ruby Sword"
    material: diamond_sword
    texture: ruby_sword.png
    parent: item/handheld    # model parent (default: item/generated)
    hand_equipped: true      # Bedrock: larger item in hand (tool/weapon feel)
    max_stack_size: 1        # default 64
```

### Fields

| Field | Required | Description |
| --- | --- | --- |
| `display_name` | Yes | MiniMessage-formatted name. Supports `<red>`, `<gradient:#start:#end>`, `<bold>`, etc. |
| `material` | Yes | Vanilla Java base item (e.g. `paper`, `iron_ingot`, `diamond_sword`). All items with the same material share one model-override file. |
| `texture` | Yes | PNG filename inside the pack's `textures/` directory. |
| `parent` | No | Java model parent. `item/generated` (flat, default) or `item/handheld` (tool grip). |
| `hand_equipped` | No | Bedrock only — shows the item larger in the hand, like a weapon or tool. Default `false`. |
| `max_stack_size` | No | Maximum stack size. Default `64`. |

### Display names

`display_name` supports the full MiniMessage syntax used throughout GoCraft:

```yaml
display_name: "<red>Ruby"
display_name: "<gradient:#ff0000:#aa0000>Ruby Sword"
display_name: "<gold><bold>Legendary Pickaxe"
display_name: "<#A0522D>Walnut Staff"
```

On Bedrock, MiniMessage tags are stripped automatically since Bedrock uses a different text format — only the plain text is shown there.

---

## server.yml configuration

```yaml
custom_items:
  enabled: true
  packs_dir: packs          # directory containing pack sub-folders

  java:
    serve_port: 8080        # port the embedded HTTP server binds on
    public_host: ""         # your server's public IP or domain
                            # e.g. "203.0.113.5" or "play.myserver.com"
```

> **Pterodactyl note:** Open `serve_port` (default `8080`) as an additional TCP port on your egg. Set `public_host` to your server's public IP. Java clients will download the pack from `http://<public_host>:8080/<hash>.zip` automatically.

If you already have `resource_pack.java.url` set manually in `server.yml`, the custom items system will override it with the auto-generated pack URL. Remove or disable the manual entry if you want full auto-management.

---

## How it works

### Java clients

1. At startup, GoCraft reads all packs, assigns a stable **CustomModelData** (CMD) integer to each item (starting at `30100`, matching the CustomiZer convention).
2. A resource pack ZIP is generated in memory containing:
   - `assets/minecraft/models/item/<material>.json` — adds CMD overrides pointing to your custom models
   - `assets/<namespace>/models/item/<id>.json` — the item model (flat or handheld)
   - `assets/<namespace>/textures/item/<id>.png` — the texture
3. The ZIP is served over HTTP on `serve_port`. The SHA-1 hash is computed and sent alongside the URL so clients can cache the pack.
4. Players receive the pack during the configuration phase. They see custom item visuals for any item stack that has the matching CMD value set.

### Bedrock clients

1. GoCraft generates a `.mcaddon` in memory containing a **resource pack** (textures + `item_texture.json`) and a **behavior pack** (one `items/<ns>/<id>.json` per item).
2. The `.mcaddon` is injected into the Bedrock listener's pack list and pushed to clients during login.
3. Custom item identifiers and runtime IDs are registered in the **StartGame** packet's item table (`ComponentBased: true`), so Bedrock clients recognise the items without needing a full vanilla server.

### ID stability

CMD values and Bedrock runtime IDs are assigned once and persisted to `packs/.registry.yml`. They never change between restarts, so items already stored in world saves or player inventories remain valid after server updates.

```yaml
# packs/.registry.yml — auto-generated, do not edit
entries:
  mypacks:ruby:
    cmd: 30100
    bedrock_rid: 5000
  mypacks:ruby_sword:
    cmd: 30101
    bedrock_rid: 5001
```

---

## Pterodactyl setup (step by step)

1. In the Pterodactyl file manager, create a `packs/` folder at `/home/container/packs/`.
2. Inside it, create a subfolder for your pack (e.g. `mypacks/`) with an `items.yml` and a `textures/` folder.
3. Upload your PNG textures into `textures/`.
4. In `server.yml`, set:
   ```yaml
   custom_items:
     enabled: true
     packs_dir: packs
     java:
       serve_port: 8080
       public_host: "YOUR_SERVER_IP"
   ```
5. Open port `8080` (TCP) on your Pterodactyl egg's port allocations.
6. Start the server. GoCraft logs `customitems: Java pack ready` and `customitems: loaded pack`.

---

## Giving custom items to players

Custom items are standard vanilla item stacks under the hood. On Java, they are the base `material` with a `CustomModelData` NBT tag set. You can give them with the standard `/give` command using NBT:

```
/give <player> paper{CustomModelData:30100}
```

A dedicated `/zitem give <player> <namespace:id>` command is planned for a future update to avoid having to remember CMD numbers.

---

## Limitations

| Feature | Status |
| --- | --- |
| 2D flat textures | Fully supported |
| Handheld (tool/weapon) pose | Supported via `parent: item/handheld` |
| 3D custom models | Not supported — requires manual Java model JSON and a pre-built Bedrock geometry file |
| Custom crafting recipes | Not yet — planned |
| Custom block items (note-block method) | Not yet |
| Multiple packs | Supported — add as many sub-directories as you like |
| `.mcaddon` with nested sub-packs | Supported via the Bedrock pack loader |

---

## Relation to CustomiZer

GoCraft's custom item system is conceptually based on [CustomiZer](https://github.com/el211/CustomiZer) — a Spigot plugin built by the same author. The pack YAML format and CMD range (`30100+`) are intentionally compatible so existing CustomiZer pack definitions can be reused with minimal changes.

Unlike CustomiZer, GoCraft owns the full protocol layer for both editions, which means Bedrock items are registered natively in the `StartGame` packet rather than via a Geyser mapping export step.
