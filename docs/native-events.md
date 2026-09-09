# Native events: Go and Java

Register Go callbacks during `OnLoad`, and declare their event names in
`[[subscribe]]` entries in `plugin.toml`. Only cancellable native callbacks take
the existing `gocraft.EventControl`; observational callbacks take the typed
event pointer alone. A nil typed callback is rejected at registration.

```go
ctx.Events().OnPlayerChat(func(e *gocraft.PlayerChatEvent, c gocraft.EventControl) {
    if e.Message == "hidden" { c.Cancel(); return }
    if e.Message == "hello" { e.Message = "Hello from Go!" }
})
ctx.Events().OnPlayerJoin(func(e *gocraft.PlayerJoinEvent) {
    ctx.Logger().Info("joined", "player", e.Player.Username)
})
```

`Cancel()` and `Cancelled()` are the cancellation API; no second cancellation
interface is introduced. Go fields are typed snapshots. Assignments to fields
not marked mutable are local only and are never returned to the server.
Java exposes setters only on mutable values, e.g. `event.setMessage(...)`.
Java listeners request `EventControl` only for cancellable events.

| Event | Cancellable | Mutable fields | Hook semantics |
| --- | --- | --- | --- |
| `player.join` | no | none | Reachable player; targeted replay after runtime recovery |
| `player.quit` | no | none | Removal from online players; repeated removal is ignored |
| `player.chat` | yes | message | Before formatting/broadcast on either edition |
| `player.command` | yes | command | Player commands, before parsing and permission checks |
| `block.break` | yes | none | Existing hook before breaking/drops/tool wear |
| `block.place` | yes | none | Validated placement, before primary/partner writes and consumption |
| `player.damage` | yes | damage | After shield/armour checks, before resistance and absorption |
| `entity.damage` | yes | damage | Non-player hits before queue coalescing; environmental damage before application |
| `player.death` | no | none | Fatal health transition after totem resolution |
| `player.respawn` | no | none | After authoritative revival/position bootstrap |
| `player.teleport` | yes | x, y, z | Command teleports in the current dimension |
| `player.interact` | yes | none | Main-hand block/entity use; not attack or dismount |
| `inventory.click` | yes | none | Java container clicks / Bedrock canonical inventory transactions |
| `item.use` | yes | none | Main-hand use in air, before starting/consuming the item |

Block placement reports the final primary block state and the replaced block;
doors/beds/chest partners commit together behind one decision. Cancelled
placements resynchronize client predictions. Inventory cancellation restores
canonical contents; Java also clears its partial drag selection.

Bedrock multi-slot inventory transactions use `slot=-1`, `button=0`, `mode=0`;
Java uses its actual click values. This is not an identical gesture model.
Java INTERACT_AT is ignored for actions already represented by INTERACT. A
forwarded canonical intent carries an internal marker to prevent redispatch.

Damage is finite and positive; setting it to zero prevents application.
`player.damage` is not raw pre-armour damage. `entity.damage` currently reports
`queued` for queued combat/projectiles because that queue has no damage-source
kind; environmental paths use `fire`, `water`, or `poison`. Zero-damage projectile
impacts remain their existing hurt/knockback operation, not a damage event.

Chat rewritten to start with `/` stays chat. Command rewrites are reparsed and
checked against the rewritten command's permissions. Console commands do not
emit a player event. Teleport mutations must be finite and within coordinate
bounds; internal login, respawn, dimension-transfer and vehicle corrections do
not become cancellable command teleports.

## Round trips and lifecycle

The ABI's positional fields remain the common contract. The generator emits
host encoders, Go decoders/mutation collectors and Java event accessors from
`gocraft-abi/abi/v1/events.proto`. The `gocraft.abi.v1.mutable` option declares
which paths the host accepts. Mutations use the existing verdict transport;
the envelope ABI is unchanged. Nested, wrong-kind, immutable and non-finite
native mutations are ignored by the host.

Handlers in one Go plugin share the event pointer and cancellation control.
Host subscriptions retain existing priority ordering. Each following plugin
sees earlier accepted mutations; cancellation short-circuits later plugins.
Observational delivery remains asynchronous: notification completion order
across separate events is not guaranteed. Cancellation cannot affect it.
Unload detaches handlers; reloading creates fresh runtime state.

The JVM's shaped warm-up and 20 ms first-event grace remain intact. Warm-up
does not call plugin handlers. The ordinary shared event budget remains 2 ms;
plugin-owned first-use initialization may still need the bounded grace.
See [validation](native-events-validation.md) for real-process tests and timing.

## Current boundaries

There is no separate PlayerKill event: player death retains the existing cause
string, not a new structured killer API. Movement, portal/pearl teleports,
creative inventory editing and every specialised tool/bucket placement are
not fully covered by dedicated events. Block interaction can cancel the
corresponding use action. These are follow-up hooks, not alternate schemas.
