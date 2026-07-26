# 🛠️ GoCraft — Milestone 1 Complete: Status Ping

**Project:** GoCraft — Minecraft Java Edition server written entirely in Go
**Date:** 2026-07-29
**Milestone:** #1 — Handshake & Server List Ping

---

## ✅ What was accomplished

GoCraft can now be compiled into a **single native binary** (no JVM, no Java, no Paper) and a real Minecraft Java client can connect, see the server in the multiplayer list, and get a ping response — exactly like a real server.

### What works right now
- Listens on **port 25565** (the default Minecraft port)
- Accepts incoming TCP connections from any Minecraft Java Edition client
- Handles the **Handshake packet** (detects protocol version, server address, requested next state)
- Serves a proper **Server List status response** with:
  - Server name / MOTD
  - Protocol version (`1.21.4 / 769`)
  - Online player count + max players
- Responds to **Ping / Pong** packets (latency shown in the client)
- Detects login attempts and gracefully rejects them (login is Milestone 2)
- Reads config from **`server.yml`** (auto-generated with defaults on first run)
- Full **structured logging** (timestamp, level, remote address, event)

---

## 📁 Package structure (12 files)

```
GoCraft/
├── main.go                   Entry point + graceful shutdown (SIGINT/SIGTERM)
├── server.yml                YAML config (MOTD, port, max players, version…)
├── config/config.go          Config loader — writes defaults if file missing
├── protocol/
│   ├── varint.go             VarInt & VarLong encode/decode (Minecraft wire format)
│   ├── types.go              String, Long, UShort, Bool helpers
│   ├── packet.go             Packet framing + Builder pattern
│   ├── varint_test.go        Tests — round-trips, edge cases, error paths
│   └── packet_test.go        Tests — framing, sequential reads, oversize rejection
├── network/
│   ├── conn.go               Per-client connection (state machine, deadlines, mutex)
│   └── listener.go           TCP accept loop → goroutine per connection
├── handler/
│   ├── handshake.go          Reads Handshake packet, routes to Status or Login
│   └── status.go             Status JSON response + Ping/Pong exchange
└── server/server.go          Top-level Server, connection dispatch
```

---

## 🧪 Tests

**13 tests, all passing.**

| Test | What it covers |
|------|---------------|
| `TestWriteVarInt` | Encodes known values to correct bytes |
| `TestReadVarInt` | Decodes known byte sequences to correct values |
| `TestVarIntRoundTrip` | Encode → decode gives back original value |
| `TestReadVarInt_TooBig` | Rejects VarInts longer than 5 bytes |
| `TestReadVarInt_UnexpectedEOF` | EOF mid-varint returns `ErrUnexpectedEOF` |
| `TestVarIntSize` | Byte-width calculation for 8 boundary values |
| `TestVarLongRoundTrip` | 64-bit equivalent of the above |
| `TestWriteReadPacket_RoundTrip` | 5 packet types encode → decode → match |
| `TestReadPacket_InvalidLength` | Zero-length packet rejected |
| `TestReadPacket_ExceedsMaxSize` | 8 MiB limit enforced |
| `TestBuilder` | Fluent builder writes String/VarInt/Long/Bool correctly |
| `TestPacket_FrameIntegrity` | Two sequential packets don't bleed into each other |

---

## 🌍 Cross-platform builds confirmed

| Platform | Status |
|----------|--------|
| Windows (amd64) | ✅ |
| Linux (amd64) | ✅ |
| macOS (amd64) | ✅ |

---

## 🗺️ Roadmap

| # | Milestone | Status |
|---|-----------|--------|
| 1 | Handshake + Server List Ping | ✅ Done |
| 2 | Login & Authentication (RSA, Mojang auth, offline mode) | 🔜 Next |
| 3 | Configuration state (registry data, feature flags) | ⬜ |
| 4 | World I/O (region files, NBT, chunk loading/saving) | ⬜ |
| 5 | Chunk generation (flat + basic noise) | ⬜ |
| 6 | Player joins (spawn, chunk sending, player list) | ⬜ |
| 7 | Movement & chat | ⬜ |
| 8 | Blocks & inventories | ⬜ |
| 9 | Entities & physics | ⬜ |
| 10 | Scheduler & event system | ⬜ |
| 11 | Go-native plugin API | ⬜ |
| 12 | Commands & permissions | ⬜ |

---

*GoCraft is a clean-room Go implementation of the Minecraft Java Edition server protocol.
No Mojang, CraftBukkit, or Paper source code was copied. Built entirely from the public protocol specification.*
