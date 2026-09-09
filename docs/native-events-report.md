# Go event API implementation report

## Branch base

- Working branch: `feat/go-events-api`, in all five changed repositories.
- Custom-events input: `50b03b2c6cdd6b3320648020044abc9bfba049a2`.
- Latest missing-items input: `916f34dcc5086398956c4b0e7545776dfb11f905`.
- Initial combined baseline: `708d056`; final input synchronization: `57ef19d`.
- No textual merge conflicts; automatic world/server merges were reviewed.
- Both input tips remain ancestors; no changes were pushed to `missing-items`.

## Event system audit

- Existing: generated BlockBreak/PlayerJoin on Go and Java; custom events,
  positional ABI, EventControl, priorities, permissions, lifecycle and warm-up.
- Missing: most fundamental native gameplay events and typed Go scalar mappings.
- Incomplete: Go and Java native mutation collection and host application.
- Shared source: ABI `events.proto` and `options.proto`, not parallel schemas.
- Full audit, baseline notes and commands: [validation](native-events-validation.md).

## Implemented

Native typed mutation/cancellation round trips, schema-driven code generation,
gameplay hooks for both client editions, real-process tests, examples and docs.
Fixed health-tracker history scans and warmed JVM mutation serialization without
replacing the existing event bus, custom-event codec or transport envelope.

## Go API changes

Typed event pointers retain the existing EventControl on cancellable callbacks.
Observational typed callbacks accept only the event pointer; OnPlayerJoin users
must remove the old control argument. Snapshot assignments stay local, while
explicitly mutable fields return through the existing positional verdict.
Typed nil handlers are rejected. See [API and semantics](native-events.md).

## Events added

BlockPlace, PlayerQuit, PlayerChat, PlayerCommand, PlayerDamage, PlayerDeath,
PlayerRespawn, PlayerTeleport, PlayerInteract, InventoryClick, ItemUse and
EntityDamage: 12 additions, 14 native event types including the existing two.
Only message, command, damage and teleport destination values are mutable.

## Missing-items integration

Preserved current placement/partner blocks, inventory reconciliation, boats,
combat/absorption/totems, projectile queues and dimension simulation. Added hooks
before authoritative mutation and consumption, with prediction resynchronization
on cancellation. The final merge also retained new villager door navigation,
bed-head claims and stale bed-occupancy fixes. Existing gameplay tests still pass.

## Tests

GoCraft, ABI and SDK `go test ./...`; JVM Gradle build/tests; generation of all
three targets with no generated diff; Go and Java example builds/bundles.
Focused race suite: core/plugin, runtime/goplugin and runtime/jvm.
Coverage includes registration, ordering/shared state, codecs, all 14 SDK schema
mappings, invalid/unknown payloads, cancellation/mutation, permissions,
observational events, duplicate prevention and unload/reload. Real Go and Java
child processes verify IPC mutation/cancellation; gameplay tests exercise packet
and intent paths. The JVM integration test requires local fixture paths and skips
explicitly without them; it was also run with the real fixture.

## Benchmarks

[Full measurements](native-events-benchmarks.md) separate cold startup, first,
warmed and 100/500/600/1000-event batches plus a host-only control.
Warm batch averages: Go 128–172 us/event, Java 119–233 us/event. Outliers exceed
2 ms; fresh warmed JVM first dispatch used the existing 20 ms cold grace.
These measurements do not establish a worst-case latency guarantee.

## Java compatibility

The same ABI schema generates both language APIs. Native Java setters return
the same allowed positional mutations as Go fields. Custom events, priorities,
EventControl, shaped warm-up and bounded first-event grace remain in place.

## Files / repos changed

- GoCraft: generator, core/plugin, core/player/world/game/intent, Java handlers,
  server gameplay hooks, runtime tests/benchmarks, module pins and event docs.
- gocraft-abi: source options/events and regenerated wire representations.
- gocraft-api-go: generated typed API, decoders, mutation collection, tests/docs.
- gocraft-jvm: generated events, native verdicts, warm-up and integration fixture.
- gocraft-plugin-examples: Go/Java chat mutation/cancellation and build instructions.
- Existing custom-events gocraft-cli reused without source modifications.

## Remaining event work

No separate PlayerKill/structured killer API; no complete movement, portal/pearl,
creative inventory or specialised placement coverage. No reusable full-bot
harness was found: an actual Minecraft login/action test and live two-example
cross-runtime session remain. See [boundaries](native-events.md#current-boundaries).

## Potential problems / technical debt

Windows latency tails need profiling and Linux confirmation. Observational
events remain asynchronous. Queued entity damage lacks structured source data.
Feature ABI/SDK commits are pinned; publish coordinated releases before stable
consumption. Existing JVM local publications report duplicate coordinates.
User IDE changes were preserved; implementation commits obey the 100-line cap.
