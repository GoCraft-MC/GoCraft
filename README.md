<p align="center">
  <img src="gocraftpng.png" alt="GoCraft — Minecraft server rewritten in Go" width="100%">
</p>

<h1 align="center">GoCraft</h1>

<p align="center">
  An experimental Minecraft: Java Edition server implementation built from scratch in Go.
</p>

> [!WARNING]
> GoCraft is early experimental software. It is **not production-ready**, does not provide a complete playable world, and should not be exposed as a public server. Expect incomplete protocol handling, breaking changes, and data-model changes during development.

## Overview

GoCraft is a native Go implementation of a Minecraft server. It is being developed from scratch around a protocol-independent game core, with edition-specific network adapters at the boundary. It is not a Paper fork, does not use the JVM, and does not currently replace Paper or another full Minecraft server.

The current implementation focuses on the connection lifecycle for Minecraft: Java Edition 1.21.4: status discovery, authentication, configuration, entry into the play state, and connection keep-alives. World data and complete gameplay are not implemented.

## Compatibility

| Client | Current status |
| --- | --- |
| Minecraft: Java Edition 1.21.4 | Active development target |
| Java protocol 769 | Implemented target |
| Other Java Edition versions | Not supported |
| Minecraft: Bedrock Edition | Planned; no adapter exists yet |

Changing `version_name` or `protocol_version` in `server.yml` changes the advertised status metadata; it does not add protocol compatibility.

## Implemented

- Native Go entry point and executable
- TCP listener, per-connection handling, and graceful process shutdown
- Minecraft packet framing, VarInt/VarLong encoding, UUIDs, and common wire types
- Handshake routing to status or login state
- Server-list status response with MOTD, version, and player limits
- Ping/pong latency exchange
- Offline-mode login with deterministic offline UUIDs
- Online-mode authentication through the Mojang session server
- RSA key exchange and AES-128-CFB8 encrypted connections
- Java configuration state:
  - known-packs negotiation for `minecraft:core` 1.21.4
  - `minecraft:brand` plugin message
  - vanilla feature flags and configuration completion
- Entry into the Java play state:
  - play login, abilities, spawn position, and initial position
  - player tab-list entry and center-chunk marker
  - teleport confirmation
  - periodic keep-alive requests and response validation
- Protocol-independent player, spatial, and online-player registry types
- YAML configuration with defaults and basic validation
- Structured logging through Go's `log/slog`
- Automated authentication, cryptography, packet, and VarInt tests

### Not implemented

GoCraft does not yet send world or chunk data, process movement or chat, implement blocks, inventories, entities, commands, permissions, or provide complete gameplay. It does not support Paper plugins, Bedrock clients, or cross-play.

## Architecture

```text
                         ┌──────────────────────────┐
Java Edition client ───▶ │ Java protocol adapter    │
                         │ network/auth/handlers    │
                         └────────────┬─────────────┘
                                      │
                         ┌────────────▼─────────────┐
                         │ Protocol-independent     │
                         │ GoCraft core             │
                         │ players/game/spatial     │
                         └────────────▲─────────────┘
                                      │
                         ┌────────────┴─────────────┐
Bedrock client ─ ─ ─ ─ ▶ │ Future Bedrock adapter  │
                         │ not implemented          │
                         └──────────────────────────┘
```

- **GoCraft core (`core/`)** owns edition-neutral player, game-registry, and spatial models. It does not import Java- or Bedrock-specific packages.
- **Java adapter (`java/`)** implements the current TCP protocol, packet handling, login authentication, encryption, configuration, and limited play-state lifecycle.
- **Bedrock adapter (`bedrock/`)** is currently a documentation-only placeholder. UDP/RakNet, Bedrock authentication, packet translation, and cross-play remain future work.
- **Server layer (`server/`)** wires configuration, the core, and the Java adapter into the executable.

## Development status

| Milestone | Status | Scope |
| --- | --- | --- |
| 1 — Handshake and status ping | Complete | Handshake, server-list response, ping/pong, YAML configuration |
| 2 — Login and authentication | Complete | Offline and online login, Mojang session verification, RSA and AES-CFB8 |
| 3 — Configuration and play-state entry | Complete | Known packs, feature flags, initial play packets, teleport confirmation, keep-alive |
| World, chunks, and gameplay | Not implemented | World storage/generation, chunk delivery, movement, chat, blocks, inventories, entities |
| Go-native plugin API | Planned | Event, scheduler, command, permission, and extension APIs |
| Bedrock adapter and cross-play | Future work | RakNet/UDP transport, Bedrock login, and translation through the shared core |

Detailed records for the completed milestones are kept in [`logs/`](logs/).

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

### Linux

```bash
git clone https://github.com/el211/GoCraft.git
cd GoCraft
go mod download
go test ./...
go build -o gocraft .
```

### macOS

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

### Windows PowerShell

```powershell
.\gocraft.exe
```

### Linux

```bash
./gocraft
```

### macOS

```bash
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
│   └── spatial/spatial.go     # Position and rotation types
├── java/
│   ├── auth/                  # Login crypto, UUIDs, Mojang sessions
│   ├── handler/               # Handshake, status, login, config, play
│   ├── network/               # TCP listener and client connections
│   └── protocol/              # Framing, packets, VarInts, wire types
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

A Go-native plugin API is planned, but **no plugin system is implemented today**. The intended direction includes events, scheduling, commands, permissions, and extension points built for GoCraft's own core. Paper, Bukkit, and Spigot plugin compatibility is not currently supported and should not be assumed.

## Bedrock and cross-play plans

Bedrock support is future work. The planned design is a separate Bedrock protocol adapter that translates Bedrock connections into the same canonical core state used by the Java adapter. The current `bedrock` package contains only design documentation—there is no RakNet listener, Bedrock login, packet implementation, or working cross-play.

## Contributing

GoCraft is still establishing its protocol and core boundaries. Before submitting a change:

1. Open an issue or discussion for large features or architecture changes.
2. Keep edition-independent code in `core/` and protocol-specific behavior in its adapter.
3. Do not claim compatibility without a test or reproducible client trace.
4. Add or update tests for protocol encoding, authentication, and state transitions.
5. Run `go fmt ./...`, `go test ./...`, and `go build ./...`.
6. Keep pull requests focused and document any protocol version assumptions.

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
