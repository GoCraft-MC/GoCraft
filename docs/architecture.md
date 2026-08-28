# Architecture

GoCraft is built around a protocol-independent game core with edition-specific network adapters at the boundary.

## Overview

```
Java Edition client ───▶  java/ adapter
                                │
                         ┌──────▼──────┐
                         │   core/     │  ← no Java or Bedrock imports
                         │  World      │
                         │  Entity     │
                         │  Player     │
                         └──────┬──────┘
                                │
Bedrock client ────────▶  bedrock/ adapter
```

- **`core/`** owns the edition-neutral game state: blocks, chunks, world, entities, players, inventories, and spatial types. It never imports `java/` or `bedrock/`. This is enforced by a compile-time architecture test in `core/world/arch_test.go`.
- **`java/`** reads from `core/` and produces native Java Edition packets: TCP framing, login auth, encryption, chunk encoding, and play-state management. It does not know Bedrock exists.
- **`bedrock/`** uses RakNet/UDP and optional Xbox authentication, translates canonical chunks to Bedrock block hashes, posts gameplay intents to the core, and synchronizes shared state back to every Bedrock session.
- **`server/`** wires configuration, the core, and the active adapters into the executable.

## Block identity

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
state ID      ID
```

This means only the encoder packages need to change when updating Java versions or adding new Bedrock support; `core/` is untouched.

## Registry abstraction

Known-packs negotiation and registry delivery are behind a `registry.Provider` interface:

```go
type Provider interface {
    Packs() []Pack
    SendRegistries(conn *network.ClientConn) error
}
```

`VanillaProvider` uses the Known-Packs shortcut (zero registry packets for vanilla 1.21.4). A future `ExplicitProvider` will send full registry data for custom dimensions and additional Java versions.

## Bedrock adapter

GoCraft is **not** a protocol translator like Geyser. The Bedrock adapter does not consume Java packets and re-encode them for Bedrock clients. Both adapters independently read from the same canonical game state and produce their own native wire format:

```
                    ┌─────────────────────────────┐
                    │         core/               │
                    │  World · Entity · Inventory │  ← no Java, no Bedrock
                    └────────────┬────────────────┘
                                 │ canonical state
               ┌─────────────────┴──────────────────┐
               ▼                                    ▼
   ┌───────────────────────┐          ┌───────────────────────┐
   │     java/ adapter     │          │   bedrock/ adapter    │
   │  native Java packets  │          │  native Bedrock packets│
   │  Java state IDs       │          │  Bedrock runtime IDs  │
   └───────────────────────┘          └───────────────────────┘
         Java client                       Bedrock client
```

## Project structure

```
GoCraft/
├── bedrock/
│   ├── listener.go         # RakNet listener, login, Bedrock play loop
│   └── world/              # Canonical block-hash and sub-chunk encoder
├── config/
│   └── config.go           # YAML loading, defaults, and validation
├── core/
│   ├── entity/             # Canonical Entity type and thread-safe registry
│   ├── game/game.go        # Edition-neutral player registry + shared entity IDs
│   ├── permission/         # Group/user permission model and manager
│   ├── player/             # Canonical Player model (position, inventory, game mode)
│   ├── spatial/            # Position and rotation types
│   └── world/
│       ├── block.go        # Block{Namespace, Name, Properties}
│       ├── chunk.go        # Sections, chunks, block entities
│       ├── generator.go    # Seeded Overworld/Nether/End terrain generator
│       ├── storage.go      # Storage interface (disk / memory)
│       ├── world.go        # Concurrent world cache with dirty-chunk tracking
│       └── arch_test.go    # Fails build if core/ imports java/
├── customitems/
│   ├── pack.go             # Item definition structs
│   ├── registry.go         # Persistent CMD / Bedrock runtime ID assignments
│   ├── manager.go          # Pack loader — scans packs/, assigns IDs
│   ├── javapack.go         # Java resource pack ZIP generator
│   ├── javaserver.go       # Embedded HTTP server for Java pack delivery
│   ├── bedrockpack.go      # Bedrock .mcaddon generator (RP + BP)
│   └── bedrockitems.go     # StartGame ItemEntry list builder
├── java/
│   ├── auth/               # Login crypto, UUIDs, Mojang sessions
│   ├── handler/            # Login/play handlers, commands, recipes, crafting
│   ├── network/            # TCP listener and client connections
│   ├── protocol/           # Framing, packets, VarInts, wire types
│   ├── registry/           # Provider interface + VanillaProvider
│   └── world/
│       ├── anvil/          # Anvil region-file I/O, NBT, atomic saves
│       ├── blocks.go       # StateID() accessor
│       ├── items.go        # ItemID/ItemName + block-placement helpers
│       ├── chunk.go        # Java chunk encoder (PalettedContainer, heightmaps)
│       └── sender.go       # Chunk burst sender
├── internal/
│   ├── gamedata/           # go:embed declarations for JSON data files
│   │   └── java/1.21.4/    # blocks.json, registries.json, recipes.json
│   └── protocoldata/       # MustCB/MustSB packet ID resolver
│       └── java/1.21.4/    # Packet ID JSON files per protocol state
├── server/
│   ├── server.go           # Core + adapter orchestration
│   ├── chatformat.go       # Chat format loader (configuration/chatformat.yml)
│   ├── glyphs.go           # Glyph map loader (configuration/glyphs.yml)
│   └── resourcepack.go     # Bedrock pack loader (.mcpack / .mcaddon)
├── docs/                   # Documentation pages
├── logs/                   # Milestone development records
├── main.go                 # Executable entry point
└── server.yml              # Runtime configuration
```

## Custom items system

The `customitems/` package implements GoCraft's cross-edition custom item system. See [Custom Items](custom-items.md) for the full guide.

```
packs/mypacks/items.yml
       │
customitems.Load()
       ├── assigns CMD integers (Java)      → packs/.registry.yml (persistent)
       ├── assigns runtime IDs (Bedrock)    → packs/.registry.yml (persistent)
       ├── BuildJavaPack() → ZIP in memory  → HTTP server on :8080
       ├── BuildBedrockPack() → mcaddon     → bedrock/listener pack list
       └── BedrockItemEntries()             → StartGame.Items (appended to vanilla registry)
```
