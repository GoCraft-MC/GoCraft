<p align="center">
  <img src="gocraftpng.png" alt="GoCraft — Minecraft server rewritten in Go" width="100%">
</p>

<h1 align="center">GoCraft</h1>

<p align="center">
  A native-Go Minecraft server built from scratch around an edition-agnostic core.
</p>

> [!WARNING]
> GoCraft is early experimental software. It is **not production-ready** and should not be exposed as a public server. Expect breaking changes and data-model changes during development.

## Overview

GoCraft is a native Go implementation of a Minecraft server written from scratch. It is built around a protocol-independent game core with edition-specific network adapters at the boundary. It is not a Paper fork, does not use the JVM, and is not a drop-in replacement for an existing server.

A vanilla Minecraft: Java Edition 1.21.4 client can connect, authenticate, complete configuration, enter the play state, and see a flat world of stone at Y=63. Movement, chat, and gameplay are not yet implemented.

## Compatibility

| Client | Current status |
| --- | --- |
| Minecraft: Java Edition 1.21.4 | Active development target |
| Java protocol 769 | Implemented |
| Other Java Edition versions | Not supported |
| Minecraft: Bedrock Edition | Planned; no adapter exists yet |

Changing `version_name` or `protocol_version` in `server.yml` changes the advertised status metadata only; it does not add protocol compatibility.

## Implemented

- Native Go entry point and executable
- TCP listener, per-connection goroutines, and graceful process shutdown
- Minecraft packet framing, VarInt/VarLong encoding, UUIDs, and common wire types
- Handshake routing to status or login state
- Server-list status response with MOTD, version, and player limits
- Ping/pong latency exchange
- Offline-mode login with deterministic offline UUIDs
- Online-mode authentication through the Mojang session server
- RSA key exchange and AES-128-CFB8 encrypted connections
- Java configuration state:
  - Known-packs negotiation via a `registry.Provider` interface (`VanillaProvider` uses the vanilla 1.21.4 shortcut; future providers can send full registry data for custom content or Bedrock translation)
  - `minecraft:brand` plugin message, vanilla feature flags, and configuration completion
- Entry into the Java play state:
  - Login, abilities, default spawn, tab-list entry, position sync, center-chunk marker
  - Teleport confirmation
  - Periodic keep-alive requests and response validation
- **Canonical world layer** (`core/world`):
  - Edition-agnostic `Block` type with `Namespace`, `Name`, and `Properties` — no Java or Bedrock IDs in the core
  - Palette-based `Section` and `Chunk` types; 24 sections per column (Y=−64 to 319)
  - `FlatGenerator` producing a single layer of stone at Y=63
  - Concurrent `World` cache with on-demand chunk generation
  - Architecture test that fails at compile time if any `core/` package imports `java/`
- **Java chunk encoding** (`java/world`):
  - Java 1.21.4 global block state ID registry (hardcoded; data-driven in a future milestone)
  - `Block → Java state ID` lookup at the adapter boundary — the core never touches Java IDs
  - Network-NBT heightmap encoding (root compound without name, 1.20.2+ format)
  - `PalettedContainer` encoder: indirect palette, ≥4 bits/entry, no-overflow packing
  - Level Chunk With Light packet (0x27) with full sky-light data for all 26 sections
  - `Sender.SendChunksAround`: 7×7 initial chunk burst after teleport confirmation
- Protocol-independent player, spatial, and online-player registry types
- YAML configuration with defaults and basic validation
- Structured logging through Go's `log/slog`
- Automated tests for authentication, cryptography, packet framing, VarInt encoding, and architecture isolation

### Not implemented

Movement, chat, block interaction, inventories, entities, commands, permissions, and complete gameplay are not implemented. Paper plugin compatibility, Bedrock clients, and cross-play are not supported.

## Architecture

```text
                         ┌──────────────────────────┐
Java Edition client ───▶ │ Java protocol adapter    │
                         │ java/network             │
                         │ java/handler             │
                         │ java/world  ─┐           │
                         │ java/registry│           │
                         └──────────────┼───────────┘
                                        │ canonical Block / Chunk
                         ┌──────────────▼───────────┐
                         │ GoCraft core             │
                         │ core/world               │  ← no Java or Bedrock imports
                         │ core/player              │
                         │ core/game                │
                         │ core/spatial             │
                         └──────────────┬───────────┘
                                        │ canonical Block / Chunk
                         ┌──────────────▼───────────┐
Bedrock client ─ ─ ─ ─ ▶ │ Future Bedrock adapter  │
                         │ bedrock/world (planned)  │
                         └──────────────────────────┘
```

### Block identity

The canonical block type carries no edition-specific IDs:

```go
// core/world — shared by all adapters
type Block struct {
    Namespace  string            // "minecraft"
    Name       string            // "stone", "grass_block", …
    Properties map[string]string // {"snowy": "false"}, nil = default state
}
```

Edition-specific IDs are resolved entirely at the adapter boundary:

```
Canonical Block
       │
┌──────┴──────┐
▼             ▼
Java global   Bedrock runtime
state ID      ID (future)
```

This means only the encoder packages need to change when updating Java versions or adding Bedrock support; `core/` is untouched. The architecture test in `core/world/arch_test.go` enforces this by failing the build if any `core/` file imports `GoCraft/java`.

### Registry abstraction

Known-packs negotiation and registry delivery are behind a `registry.Provider` interface:

```go
type Provider interface {
    Packs() []Pack
    SendRegistries(conn *network.ClientConn) error
}
```

`VanillaProvider` uses the Known-Packs shortcut (zero registry packets for vanilla 1.21.4). A future `ExplicitProvider` will send full registry data for custom dimensions, custom biomes, additional Java versions, and Java-to-Bedrock ID translation.

- **GoCraft core (`core/`)** owns edition-neutral block, chunk, world, player, game-registry, and spatial models. It never imports `java/` or `bedrock/`.
- **Java adapter (`java/`)** implements the TCP protocol, packet handling, login authentication, encryption, configuration, chunk encoding, and play-state lifecycle.
- **Bedrock adapter (`bedrock/`)** is a documentation-only placeholder. UDP/RakNet, Bedrock authentication, packet translation, and cross-play remain future work.
- **Server layer (`server/`)** wires configuration, the core, and the Java adapter into the executable.

## Development status

| Milestone | Status | Scope |
| --- | --- | --- |
| 1 — Handshake and status ping | Complete | Handshake, server-list response, ping/pong, YAML configuration |
| 2 — Login and authentication | Complete | Offline and online login, Mojang session verification, RSA and AES-CFB8 |
| 3 — Configuration and play-state entry | Complete | Known packs, feature flags, initial play packets, teleport confirmation, keep-alive |
| 4 — World layer and chunk streaming | Complete | Canonical Block/Chunk types, FlatGenerator, Java chunk encoding, initial chunk burst |
| 5 — Movement and dynamic chunk streaming | Complete | Movement packet handling, posToChunk floor-division, per-boundary chunk load/unload |
| 6 — Multiplayer sync | Planned | Spawn/despawn entities, broadcast position and head rotation to all sessions |
| 7 — Chat | Planned | Receive chat messages, broadcast to all players, `/` command prefix |
| 8 — Block interaction | Planned | Break/place blocks, mutate canonical World, broadcast Block Update to all players |
| 9 — World persistence | Planned | Anvil region-file loader, NBT reader, RegionLoader implementing Generator, chunk saving |
| 10 — Inventory and items | Planned | Player inventory, hotbar, Click Container, Set Held Item, item drops |
| 11 — Entity system | Planned | Canonical Entity type, entity registry, mob spawn/tick/despawn, health and damage |
| 12 — Commands | Planned | Command dispatcher, Commands packet (tab-completion tree), /gamemode /tp /give /kick |
| 13 — Data-driven registries | Planned | Load block state IDs and biome IDs from Minecraft's data-generator output (reports/blocks.json); replace hardcoded tables in java/world so both Java and Bedrock adapters share the same canonical ID resolution |
| 14 — Bedrock adapter | Future work | RakNet/UDP, Xbox auth, bedrock/world encoder using M13 registries for runtime IDs, cross-play via shared core/ |
| 15 — Go plugin API | Future work | Event bus, command registration, scheduler, permission nodes; plugins are compiled Go packages |

Detailed records for completed milestones are kept in [`logs/`](logs/).

## Requirements

- [Go](https://go.dev/) 1.23 or newer, as declared by `go.mod`
- Git, when cloning the repository
- A Minecraft: Java Edition 1.21.4 client for manual connection testing
- Network access to Mojang's session service when `online_mode: true`

No Java runtime is required.

## Clone, test, and build

### Windows PowerShell

```powershell
git clone https://github.com/el211/GoCraft.git
Set-Location GoCraft
go mod download
go test ./...
go build -o gocraft.exe .
```

### Linux / macOS

```bash
git clone https://github.com/el211/GoCraft.git
cd GoCraft
go mod download
go test ./...
go build -o gocraft .
```

## Configure

GoCraft reads `server.yml` from its working directory. The repository includes:

```yaml
host: 0.0.0.0
port: 25565
motd: A GoCraft Server
max_players: 20
version_name: 1.21.4
protocol_version: 769
online_mode: false
```

| Setting | Meaning |
| --- | --- |
| `host` | Address on which the Java TCP listener binds |
| `port` | Java Edition server port |
| `motd` | Text shown in the multiplayer server list |
| `max_players` | Advertised player limit |
| `version_name` | Advertised version name; currently `1.21.4` |
| `protocol_version` | Advertised protocol number; currently `769` |
| `online_mode` | Enables Mojang session authentication and encrypted login |

If `server.yml` is absent, GoCraft creates it with defaults. Offline mode does not verify player identities; use it only in a trusted development environment.

## Run

Run the server from the repository root so it can find `server.yml`.

```powershell
# Windows
.\gocraft.exe
```

```bash
# Linux / macOS
./gocraft
```

Stop the server with <kbd>Ctrl</kbd>+<kbd>C</kbd>. The default listener is `0.0.0.0:25565`; connect a Java Edition 1.21.4 client to `localhost:25565` when testing locally.

## Project structure

```text
GoCraft/
├── bedrock/
│   └── doc.go                 # Future Bedrock adapter placeholder
├── config/
│   └── config.go              # YAML loading, defaults, and validation
├── core/
│   ├── game/game.go           # Edition-neutral online-player registry
│   ├── player/player.go       # Canonical player model
│   ├── spatial/spatial.go     # Position and rotation types
│   └── world/
│       ├── block.go           # Block{Namespace, Name, Properties} — no edition IDs
│       ├── chunk.go           # Section and Chunk with palette-based block storage
│       ├── generator.go       # Generator interface and FlatGenerator
│       ├── world.go           # Concurrent world cache
│       └── arch_test.go       # Fails build if core/ imports java/
├── java/
│   ├── auth/                  # Login crypto, UUIDs, Mojang sessions
│   ├── handler/               # Handshake, status, login, config, play
│   ├── network/               # TCP listener and client connections
│   ├── protocol/              # Framing, packets, VarInts, wire types
│   ├── registry/              # Provider interface + VanillaProvider
│   └── world/                 # Java block state IDs, chunk encoder, Sender
├── logs/                      # Milestone development records
├── server/
│   └── server.go              # Core and Java adapter orchestration
├── go.mod
├── go.sum
├── gocraftpng.png             # README banner
├── main.go                    # Executable entry point
└── server.yml                 # Runtime configuration
```

## Plugin API plans

A Go-native plugin API is planned, but **no plugin system is implemented today**. The intended direction includes events, scheduling, commands, permissions, and extension points built for GoCraft's own core. Paper, Bukkit, and Spigot plugin compatibility is not supported and should not be assumed.

## Bedrock and cross-play plans

Bedrock support is planned for Milestone 14, after the data-driven registry layer (M13) is in place.

The canonical `Block` and `Chunk` types in `core/` carry no edition-specific IDs. M13 will load block state and biome mappings from Minecraft's own data-generator output (`reports/blocks.json`), giving both the Java adapter and the future Bedrock adapter a single shared ID-resolution path. The Bedrock encoder (`bedrock/world`) will use the same registry infrastructure to map canonical `Block` values to Bedrock runtime IDs — no separate hardcoded tables needed.

This sequencing means the Bedrock adapter is a transport and translation layer (RakNet/UDP, Xbox auth, Sub Chunk packet format) rather than a second hand-maintained block table. Cross-play — Java and Bedrock clients sharing the same world — follows naturally because both adapters read from the same `core/world.World`.

The current `bedrock/` package contains only design documentation. No RakNet listener, Bedrock login, packet encoding, or working cross-play exists yet.

## Contributing

GoCraft is still establishing its protocol and core boundaries. Before submitting a change:

1. Open an issue or discussion for large features or architecture changes.
2. Keep edition-independent code in `core/` and protocol-specific behavior in its adapter.
3. Never store edition-specific IDs (Java state IDs, Bedrock runtime IDs) in `core/` types.
4. Do not claim compatibility without a test or reproducible client trace.
5. Add or update tests for protocol encoding, authentication, and state transitions.
6. Run `go fmt ./...`, `go test ./...`, and `go build ./...`.
7. Keep pull requests focused and document any protocol version assumptions.

Please avoid adding generated binaries, credentials, player data, or private server logs.

## License

**Copyright © 2026 Oreo Studios — All Rights Reserved**

GoCraft is a proprietary software project developed and maintained by **Oreo Studios**.

This repository is intentionally public so the community can follow the development of the project. **Public visibility does not grant permission to copy, redistribute, modify, commercialize, or create derivative works from the source code.**

Unless you have received **prior written permission** from Oreo Studios, you may **not**:

* Copy substantial portions of the source code.
* Redistribute or publish modified versions.
* Create commercial or non-commercial derivatives.
* Rebrand or represent GoCraft as your own project.
* Use GoCraft code in another server software.

If you are interested in contributing to GoCraft or collaborating with Oreo Studios, we would love to hear from you. Please open an issue or contact us before starting any work.

**Company Information**

**Oreo Studios — Web & Game Development Studio**

SIREN: **993 823 459**
SIRET: **993 823 459 00017**
APE Code: **62.01Z**
Entrepreneur individuel — France

Website: https://oreostudios.fr

Minecraft is a trademark of Microsoft. GoCraft is an independent project and is not affiliated with or endorsed by Mojang Studios or Microsoft.

## Credits and acknowledgements

- The [Go project](https://go.dev/) and its standard library
- [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) for YAML configuration
- Mojang Studios and Microsoft for Minecraft and its Java Edition protocol ecosystem
- Nukkit, Dragonfly, and gophertunnel, referenced in the repository's future Bedrock planning notes

GoCraft is an independent, unofficial project and is not affiliated with or endorsed by Mojang Studios or Microsoft. Minecraft is a trademark of Microsoft.
