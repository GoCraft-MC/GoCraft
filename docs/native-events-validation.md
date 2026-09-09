# Native event validation

See the [implementation report](native-events-report.md) and the
[measured benchmark results](native-events-benchmarks.md) for the final status.

## Branch inputs and audit

`feat/go-events-api` was created from custom-events
`50b03b2c6cdd6b3320648020044abc9bfba049a2`, then merged with missing-items
`31922d7f4bc7565467eeff3363559d02b994087f` before implementation. The original
merge was clean. Baseline Go/Java tests ran before new event code. Go dependency
pins had to be aligned with the existing custom-events ABI/SDK; Java's schema
fingerprint needed CRLF normalization on Windows, not a schema rewrite.

Final synchronization also merged missing-items
`916f34dcc5086398956c4b0e7545776dfb11f905` (six villager/door/bed commits) in
`57ef19d`. Both merges auto-combined `server/server.go` and core world changes
without textual conflicts. Full Go tests passed again after synchronization.

The audit found native `block.break` and `player.join`, generated host/SDK/Java
representations, custom event codecs, permission injection, priorities,
bounded cancellation, unload/recovery tests and handler-free JVM warm-up.
Native mutations were not collected by the Go SDK or applied by the host;
Java native dispatch also omitted their return path. The common generator
had no native mutable-field option and lacked several Go scalar mappings.
The new schema and generated targets close those gaps without replacing the
custom event registry, `EventControl`, transport or command tree generator.

Companion work lives on `feat/go-events-api` in `gocraft-abi`, `gocraft-api-go`,
`gocraft-jvm`, and `gocraft-plugin-examples`. The custom-events CLI was reused;
its source was not changed. Event generation still starts from the common ABI
schema via `buf.gen.events.yaml`; use `lang=go`, `lang=gosdk`, `lang=java` for the
three targets, changing only output directories for non-sibling checkouts.

## Repeatable checks

Run `go test ./...` in GoCraft, the ABI and the Go SDK. Run the JVM repository's
existing `./gradlew build`. `cmd/protoc-gen-gocraft` tests compare the checked-in
host output with the pinned schema and generate all targets from that schema.
Regenerate ABI protobuf output with `buf generate` inside `gocraft-abi`; inspect
all three repositories after event generation. Never patch generated output.

`runtime/goplugin/TestGameplayChatRoundTripThroughNativeProcessAndReload` drives
the shared gameplay chat filter through a real child process and socket. It
checks returned mutation, cancellation, detachment and reload. Server/handler
tests drive real packet/intent handlers for placements, inventories,
interactions, command permissions, player damage/death/respawn, and entity
damage across dimensions. Existing missing-items tests remain in the suite.

For real JVM tests, generate a fixture using the existing Gradle tests:

```sh
# From gocraft-jvm, with an absolute output directory:
GOCRAFT_EVENT_FIXTURE_DIR="$PWD/gocraft-runtime-jvm/build/native-fixture" ./gradlew build
# From GoCraft, using the resulting paths:
GOCRAFT_EVENT_RUNTIME_JAR="/path/gocraft-runtime.jar" \
GOCRAFT_EVENT_FIXTURE="/path/native-fixture/BenchmarkPlugin.gcpkg" \
go test ./runtime/jvm -run TestNativeEventRoundTrip -v
```

Those tests use a compiled fixture plugin and the real JVM process, not a fake
Java peer. They skip explicitly when paths are absent. No production login or
Minecraft account is needed. Java tests separately verify a warm-up produces
no effects/mutations and does not execute the handler.

The example repository documents generation order and builds both `.gcpkg`
files with its existing Gradle/shell flow. `hello-go` / `hello-java` exercise
native message mutation; `hide-go` / `hide-java` exercise cancellation.

## Performance methodology

```sh
go test ./runtime/goplugin -run '^$' -bench '^BenchmarkNativeEvent' -benchtime=20x -v
# With the same JVM environment variables:
go test ./runtime/jvm -run '^$' -bench '^BenchmarkNativeEventJVM' -benchtime=20x -v
```

Cold process load, first event, shaped JVM warm-up, warmed single events and
batches of 100/500/600/1000 are measured separately. `ns/op` is a whole batch;
`ns/event` is the average per event. The host-only case excludes IPC, SDK and
handler execution. IPC cases include a tiny handler that rewrites one string.
They are not representative of arbitrary plugin execution time.

Only the benchmark uses a 1-second measurement deadline to retain slow samples;
`over2ms_percent` counts those beyond the unchanged 2-ms production target.
Functional tests keep the real budget and cold grace. These are local Windows
measurements, not a latency guarantee or proof that 1000 IPC calls fit one tick.
The benchmark exposed a history-length scan in plugin health accounting;
rolling counters now preserve the exact window/disable semantics in amortized
constant time. JVM warm-up also exercises mutation encoding without invoking
plugin handlers or publishing synthetic state.

## Remaining validation and scope

No reusable full Minecraft bot harness was present. Packet/intent tests and
real runtime IPC tests are provided; a complete bot login/action scenario and
an in-game two-example cross-runtime session remain to be run. Observational
events retain asynchronous delivery and do not guarantee cross-event completion
order. See the event guide for specialised gameplay paths not yet hooked.
