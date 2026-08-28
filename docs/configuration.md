# Configuration

GoCraft reads `server.yml` from its working directory. A default file is created automatically on first run.

## Full reference

```yaml
java_enabled: true
host: 0.0.0.0
port: 25565
motd: A GoCraft Server
server_icon: server-icon.png
max_players: 20
version_name: 1.21.4
protocol_version: 769
online_mode: false
villagers: true
difficulty: normal
default_gamemode: survival

mob_spawning:
  hostile: 35
  passive: 16
  ambient: 15
  axolotls: 5
  underground_water_creatures: 5
  water_creatures: 5
  water_ambient: 20

operators: []

whitelist:
  enabled: false
  players: []

resource_pack:
  java:
    enabled: false
    url: ""       # HTTPS URL to the .zip resource pack
    hash: ""      # SHA-1 hex of the zip
    forced: false
    prompt: ""    # MiniMessage text shown before the accept dialog
  bedrock:
    enabled: false
    paths: []     # local paths to .mcpack, .zip, or .mcaddon files
    forced: false

custom_items:
  enabled: true
  packs_dir: packs
  java:
    serve_port: 8080
    public_host: ""   # your server's public IP or domain

permission_editor:
  enabled: true
  editor_url: https://el211.github.io/GoCraft/editor
  bytebin_url: https://bytebin.lucko.me

combat:
  attack_cooldown: false
  knockback_horizontal: 0.4
  knockback_vertical: 0.4

item_tooltips:
  show_durability: true
  show_attributes: true
  hide_vanilla_attributes: true

clear_lag:
  enabled: false
  interval_seconds: 300
  minimum_entity_age_seconds: 30
  warning_seconds: [60, 30, 10, 5, 4, 3, 2, 1]
  warning_message: "[ClearLag] Removing old entities in {seconds}s"
  complete_message: "[ClearLag] Removed {count} old entities"
  remove:
    dropped_items: true
    experience_orbs: true
    projectiles: true
    primed_tnt: true
    falling_blocks: true
    boats: false
    passive_mobs: false
    hostile_mobs: false

world_storage: disk
world_dir: world
world_seed: 0
view_distance: 8
pregenerate_radius: 8
max_cached_chunks: 256

bedrock:
  enabled: false
  address: 0.0.0.0:19106
  online_mode: true

debug:
  startup_registry: false
  environment_overrides: false
  world_loading: false
  mob_spawning: false
  autosaves: false
  entity_events: false
  entity_tick_overruns: false
  bedrock_catalogues: false
  bedrock_login: false
  bedrock_chunks: false
  bedrock_inventory: false
  profiling: false
```

## Setting reference

| Setting | Description |
| --- | --- |
| `java_enabled` | Enables the Java Edition TCP listener |
| `host` | Bind address for the Java TCP listener |
| `port` | Java Edition server port (default `25565`) |
| `motd` | Text shown in the multiplayer server list |
| `server_icon` | Java server-list image; PNG files are resized to 64×64 automatically |
| `max_players` | Advertised player limit |
| `online_mode` | Requires Mojang session authentication |
| `villagers` | Spawns village residents and iron-golem guards |
| `difficulty` | `peaceful` / `easy` / `normal` / `hard` |
| `default_gamemode` | `survival` / `creative` / `adventure` / `spectator` |
| `mob_spawning.*` | Natural-mob caps per 17×17 chunk area; `0` disables a category |
| `combat.attack_cooldown` | `false` = legacy rapid-attack feel |
| `combat.knockback_*` | Knockback strength `0`–`4` |
| `world_storage` | `disk` = persistent Anvil files; `memory` = ephemeral |
| `world_dir` | Anvil world folder (disk mode only) |
| `world_seed` | Signed 64-bit terrain seed |
| `view_distance` | Java chunk radius `2`–`32` |
| `pregenerate_radius` | Background warm-up radius `view_distance`–`64` |
| `max_cached_chunks` | RAM cache limit `128`–`65536` |
| `whitelist.*` | Shared Java/Bedrock allowlist; runtime changes persist to `whitelist.json` |
| `resource_pack.java.*` | Java resource pack pushed during configuration phase |
| `resource_pack.bedrock.paths` | Bedrock packs (`.mcpack` / `.zip` / `.mcaddon`) loaded at startup |
| `custom_items.*` | See [Custom Items](custom-items.md) |
| `permission_editor.*` | See [Permissions](permissions.md) |
| `bedrock.*` | Enables the RakNet/UDP listener and Xbox auth |
| `debug.*` | Verbose per-category logging; disable for production |

## World storage

Disk mode creates sibling folders for each dimension:

```
world/          ← Overworld (Anvil region files)
world_nether/
world_end/
```

Data from the old `world/DIM-1` and `world/DIM1` layout is migrated automatically on first startup. Chunks are autosaved every 30 seconds and flushed on clean shutdown.

Changing `world_seed` only affects **newly generated** chunks. Use a different `world_dir` when switching seeds to avoid terrain seams at old chunk boundaries.

## Environment variable overrides

All critical fields can be overridden at runtime via environment variables. Useful on Pterodactyl or Docker when YAML edits are inconvenient.

| Variable | Field |
| --- | --- |
| `GOCRAFT_JAVA_HOST` | `host` |
| `GOCRAFT_JAVA_PORT` | `port` |
| `GOCRAFT_JAVA_ENABLED` | `java_enabled` |
| `GOCRAFT_ONLINE_MODE` | `online_mode` |
| `GOCRAFT_MOTD` | `motd` |
| `GOCRAFT_SERVER_ICON` | `server_icon` |
| `GOCRAFT_MAX_PLAYERS` | `max_players` |
| `GOCRAFT_WORLD_STORAGE` | `world_storage` |
| `GOCRAFT_WORLD_DIR` | `world_dir` |
| `GOCRAFT_WORLD_SEED` | `world_seed` |
| `GOCRAFT_VIEW_DISTANCE` | `view_distance` |
| `GOCRAFT_PREGENERATE_RADIUS` | `pregenerate_radius` |
| `GOCRAFT_MAX_CACHED_CHUNKS` | `max_cached_chunks` |
| `GOCRAFT_DIFFICULTY` | `difficulty` |
| `GOCRAFT_WHITELIST_ENABLED` | `whitelist.enabled` |
| `GOCRAFT_BEDROCK_ENABLED` | `bedrock.enabled` |
| `GOCRAFT_BEDROCK_ADDR` | `bedrock.address` |
| `GOCRAFT_BEDROCK_ONLINE_MODE` | `bedrock.online_mode` |
| `GOCRAFT_PERMISSION_EDITOR_ENABLED` | `permission_editor.enabled` |
| `GOCRAFT_PERMISSION_EDITOR_URL` | `permission_editor.editor_url` |
| `GOCRAFT_PERMISSION_EDITOR_BYTEBIN` | `permission_editor.bytebin_url` |
