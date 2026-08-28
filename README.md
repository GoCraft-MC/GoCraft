<p align="center">
  <img src="gocraftpng.png" alt="GoCraft — Minecraft server rewritten in Go" width="100%">
</p>

<h1 align="center">GoCraft</h1>

<p align="center">
  A native-Go Minecraft server built from scratch around an edition-agnostic core.<br>
  Java Edition 1.21.4 · Bedrock Edition 1.26.40 · Cross-play · No JVM
</p>

<p align="center">
  <a href="https://discord.gg/Her3JEWWdj"><img src="https://img.shields.io/badge/Discord-Join%20our%20server-5865F2?style=for-the-badge&logo=discord&logoColor=white" alt="Discord"></a>
</p>

> [!WARNING]
> GoCraft is early experimental software. It is **not production-ready** and should not be exposed as a public server. Expect breaking changes during development.

---

## What is GoCraft?

GoCraft is a native Go implementation of a Minecraft server written from scratch. It is not a Paper fork, does not use the JVM, and is not a drop-in replacement for any existing server software.

Both Java and Bedrock clients connect to the same server, share the same world and entities, and see each other in real time — with no external proxy required.

## Compatibility

| Client | Status |
| --- | --- |
| Minecraft: Java Edition 1.21.4 (protocol 769) | Active development target |
| Minecraft: Bedrock Edition 1.26.40 (protocol 2168) | Beta — full cross-play support |
| Other versions | Not supported |

## Quick start

```bash
git clone https://github.com/el211/GoCraft.git
cd GoCraft
go mod download
go test ./...
go build -o gocraft .
./gocraft
```

Connect a Java 1.21.4 client to `localhost:25565`.

> **Windows:** use `go build -o gocraft.exe .` and `.\gocraft.exe`

## Documentation

| Page | Description |
| --- | --- |
| [Configuration](docs/configuration.md) | Full `server.yml` reference and environment variable overrides |
| [Custom Items](docs/custom-items.md) | Cross-edition custom item system — define once, works on Java and Bedrock |
| [Permissions](docs/permissions.md) | Group-based permission system and the GoPerm browser editor |
| [Commands](docs/commands.md) | All built-in commands with permission nodes |
| [Bedrock & Cross-play](docs/bedrock.md) | Bedrock setup, shared features, and pack support |
| [Architecture](docs/architecture.md) | Core design, adapter pattern, project structure |
| [Command Parity](docs/command-parity.md) | Gaps relative to vanilla Paper/Spigot |

## Highlights

- **No JVM** — pure Go, single binary, low memory footprint
- **Java + Bedrock cross-play** — both editions share the same canonical world and entity simulation
- **Server identity** — configurable MOTD and automatically resized Java server-list icon with an embedded default
- **Custom items** — define items in YAML, GoCraft auto-generates Java resource packs and Bedrock behavior packs at startup
- **Permission editor** — browser-based group editor via bytebin relay, no inbound port needed
- **MiniMessage** — full gradient, hex color, and glyph support in chat, prefixes, and item names
- **Resource packs** — push `.mcpack`, `.zip`, and `.mcaddon` files to Bedrock clients; serve Java packs automatically
- **Persistent world** — Anvil region files, autosaves, atomic writes, memory-mode option
- **Data-driven** — block states, item IDs, entity types, biomes, and packet IDs loaded from versioned JSON at startup; no hardcoded maps

## Development status

| Milestone | Status |
| --- | --- |
| 1–3 — Handshake, login, configuration | Complete |
| 4–5 — World, chunk streaming, movement | Complete |
| 6–7 — Multiplayer sync, chat | Complete |
| 8–10 — Block interaction, persistence, inventory | Complete |
| 11–13 — Entity system, commands, data-driven registries | Complete |
| 14 — Bedrock adapter | Beta |
| Custom items (Java + Bedrock) | In progress |
| 15 — Go plugin API | Future |

Full milestone details are in [`logs/`](logs/).

## Requirements

- Go 1.24 or newer
- A Minecraft: Java Edition 1.21.4 client for testing
- Network access to Mojang's session service when `online_mode: true`

No Java runtime is required.

## Contributing

1. Open an issue or discussion before large features or architecture changes.
2. Keep edition-independent code in `core/` — it must never import `java/` or `bedrock/`.
3. Never store edition-specific IDs (Java state IDs, Bedrock runtime IDs) in `core/` types.
4. Run `go fmt ./...`, `go test ./...`, and `go build ./...` before submitting.
5. Keep pull requests focused and document any protocol version assumptions.

## License

GoCraft is open source software licensed under the **GNU General Public License v3.0 (GPL-3.0)**.

Copyright © 2026 Oreo Studios — [oreostudios.fr](https://oreostudios.fr)

SIREN: 993 823 459 · SIRET: 993 823 459 00017 · APE: 62.01Z · Entrepreneur individuel — France

You are free to use, study, fork, and contribute to GoCraft. Any modified or derived version must also be released under GPL-3.0. See the [LICENSE](LICENSE) file for the full text.

Minecraft is a trademark of Microsoft. GoCraft is an independent project and is not affiliated with or endorsed by Mojang Studios or Microsoft.
