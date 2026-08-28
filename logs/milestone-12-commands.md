# GoCraft — Milestone 12: Commands & Permissions

**Date:** 2026-08-11
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Command dispatcher
- `java/handler/command.go`: `Dispatcher` with `Register(name, handler)` and `Dispatch(input, ctx)` methods
- Commands registered as closures in `server/server.go` — no reflection, no annotation magic
- `CommandContext` struct passed to every handler: `Player`, `Conn`, `World`, `Manager`, `TeleportTo`, `ChangeWorld`
- Unknown command sends red error message back to the sender

### Declare Commands packet
- `java/handler/commands.go`: builds `Declare Commands` (0x11) with a proper command graph
- Each registered command appears as a LITERAL node; sub-arguments declared as ARGUMENT nodes
- Client uses this for tab-completion and the `/` command suggestion UI

### Built-in commands

| Command | Permission | Description |
|---------|-----------|-------------|
| `/tp <x> <y> <z>` | `gocraft.command.tp` | Teleport to coordinates |
| `/gamemode <mode>` | `gocraft.command.gamemode` | Change own game mode |
| `/give <item> [count]` | `gocraft.command.give` | Add item to inventory |
| `/kick <player> [reason]` | `gocraft.command.kick` | Disconnect a player |
| `/stop` | `gocraft.command.stop` | Graceful shutdown |
| `/say <message>` | `gocraft.command.say` | Broadcast as server |
| `/list` | _(everyone)_ | Show online players |

### Permission system
- `core/permissions/` — group-based system loaded from `configuration/permissions.yml`
- Groups: name, weight, prefix (MiniMessage), list of permission nodes
- Players assigned to groups via `configuration/players.yml`
- `HasPermission(player, node)` — checks player's group(s) + wildcard `*` nodes
- Browser-based editor: `GoPerm` served via bytebin relay (no inbound port needed)

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Command dispatcher | ✅ M12 |
| Tab-completion graph (Declare Commands) | ✅ M12 |
| Built-in commands (tp, gamemode, give, kick, stop, say, list) | ✅ M12 |
| Group-based permissions | ✅ M12 |
| MiniMessage prefixes in chat | ✅ M12 |
| Data-driven registries | ❌ M13 |
| Bedrock support | ❌ M14 |

---

## Architecture additions

```
java/handler/
├── command.go    ← Dispatcher, CommandContext, FormatChat
└── commands.go   ← Declare Commands packet builder

core/permissions/
├── permissions.go   ← Group, HasPermission
└── loader.go        ← YAML parser for permissions.yml / players.yml

configuration/
├── permissions.yml  ← group definitions
└── players.yml      ← player → group assignments
```
