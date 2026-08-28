# GoCraft — Milestone 6: Multiplayer Sync

**Date:** 2026-08-01
**Protocol:** Minecraft Java Edition 1.21.4 (protocol 769)

---

## What was completed

### Session manager
- `java/session/manager.go`: thread-safe registry of active Java sessions
  - `Add(s *Session)`, `Remove(id uuid.UUID)`, `SnapshotAll() []*Session`
  - `ObserveMessage(text string)` — notifies registered observer (used by Bedrock bridge, M14)
- `java/session/session.go`: wraps `*network.ClientConn` + `*player.Player`

### Player list (tab list)
- On join: sends `Player Info Update` (ADD_PLAYER + UPDATE_LISTED + UPDATE_DISPLAY_NAME) to all online sessions
- On quit: sends `Player Info Remove` to all remaining sessions
- All existing players are advertised to the newly joined player at login

### Entity spawn / despawn
- On join: sends `Add Player` entity packet to every other session
- On quit: sends `Remove Entities` to every remaining session
- Entity IDs allocated from `core/game` atomic counter

### Position broadcast
- Every movement update is rebroadcast as `Entity Position And Rotation` or `Entity Teleport` (for large deltas) to all other sessions
- `Head Rotation` packet sent separately for head yaw
- No delta compression for large teleports — always sends absolute position

### Latency / ping updates
- `Player Info Update` with UPDATE_LATENCY action sent every 5 seconds for all players
- Latency measured as round-trip of the Keep-Alive exchange

---

## Current capabilities

| Feature | Status |
|---------|--------|
| Players see each other on join | ✅ M6 |
| Tab list updates on join/leave | ✅ M6 |
| Position broadcast (smooth movement) | ✅ M6 |
| Player despawn on quit | ✅ M6 |
| Chat | ❌ M7 |
| Block interaction | ❌ M8 |

---

## Architecture additions

```
java/session/
├── session.go    ← Session wrapper
└── manager.go    ← Thread-safe registry, snapshot, message observer
```
