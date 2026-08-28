# GoCraft — Milestone 7: Chat & Commands

**Date:** 2026-08-02
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Chat handling
- `java/handler/chat.go`
  - Handles `Chat Message` (0x06) — message + trailing signature fields ignored
  - Handles `Chat Command` (0x04) — command without leading `/`
  - Messages starting with `/` in Chat Message are routed to the command dispatcher
  - 256-character limit enforced (matches vanilla)
  - `buildSystemChatMessage` encodes text as Network NBT text component (required since 1.20.3)

### MiniMessage formatter
- `java/handler/minimessage.go` — full MiniMessage parser producing §-coded output:
  - Named colors (`<red>`, `<gold>`, `<dark_blue>`, …)
  - Hex colors (`<#RRGGBB>`)
  - Gradients (`<gradient:#RRGGBB:#RRGGBB>`)
  - Formatting tags (`<bold>`, `<italic>`, `<underlined>`, `<strikethrough>`, `<obfuscated>`)
  - Close tags (`</red>`, `</gradient>`, etc.) with stack-based state restore
  - Reset (`<reset>`, `<r>`)
  - Legacy `&` codes as synonym for `§`
  - Glyph tags (`<glyph:name>`) using a configured glyph map
- `EscapeMiniMessage` — escapes `<` in player messages to prevent tag injection

### Chat format config
- `configuration/chatformat.yml` — configurable per-server:
  - `format`: Java MiniMessage template with `{prefix}`, `{player}`, `{message}` placeholders
  - `bedrock_format`: fallback template using only named colors (no gradients/hex)
- Auto-generated on first run with gradient default:
  ```
  {prefix}<gradient:#5865F2:#EB459E>{player}</gradient> <white>:</white> <gray>{message}</gray>
  ```

### Bedrock-safe rendering
- `ParseMiniMessageBedrock` — same parser with `bedrockSafe` mode:
  - Gradients collapsed to first stop color via `nearestNamedColor` (Euclidean RGB distance to 16 named colors)
  - Hex `<#RRGGBB>` mapped to nearest named §-color
- Chat split: Java sessions receive Java-formatted string; Bedrock clients receive Bedrock-safe string via the listener's `BroadcastMessage`

### Command dispatcher
- `java/handler/command.go`: `Dispatcher` with `Register` and `Dispatch` methods
- `Dispatcher.FormatChat` / `FormatBedrockChat` — delegates to chat format config
- Built-in commands wired in `server/server.go` as closures

### NBT text component encoding
- `nbtTextComponent` — TAG_Compound with a single `text` TAG_String (no name, network NBT)
- `nbtLinkComponent` — includes `color: aqua`, `underlined: true`, clickEvent `open_url`
- `nbtLoreTextComponent` — italic=false override for item lore lines

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Chat broadcast (MiniMessage formatted) | ✅ M7 |
| Bedrock-safe chat formatting | ✅ M7 |
| Command dispatcher | ✅ M7 |
| Per-player chat prefix support | ✅ M7 |
| Block interaction | ❌ M8 |
| Inventory | ❌ M10 |

---

## Architecture additions

```
java/handler/
├── chat.go          ← Chat Message, Chat Command, broadcast helpers, NBT encoder
├── minimessage.go   ← MiniMessage parser, Bedrock-safe variant, gradient engine
└── command.go       ← Dispatcher, FormatChat, SetBedrockChatFormatter

server/
└── chatformat.go    ← loadChatFormat, apply, applyBedrock

configuration/
└── chatformat.yml   ← runtime config (format + bedrock_format)
```
