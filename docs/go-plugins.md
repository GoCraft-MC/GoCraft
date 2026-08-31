# Native Go plugins

Native Go plugins are experimental. Each plugin runs as its own process and
communicates with GoCraft through the versioned, protocol-neutral Plugin API.
A panic or process crash therefore does not directly crash the server.

## Install a plugin

Put its `.gcpkg` file in `plugins/` and restart GoCraft. The server discovers
bundles before opening its network listeners. A failed plugin prevents startup
instead of leaving gameplay partly protected.

Configuration defaults stored under `config/` inside the bundle are copied on
first load to `plugins/<plugin-id>/`. Existing administrator files are never
overwritten. `Context.DataDirectory()` returns that directory.

## Lifecycle

Implement `gocraft.Plugin`. The import path does not end in the package name,
so name it:

```go
import gocraft "GoCraft/gocraft-api-go"

type Plugin struct{}

func (*Plugin) OnLoad(ctx gocraft.Context) error { return nil }
func (*Plugin) OnEnable() error                  { return nil }
func (*Plugin) OnDisable() error                 { return nil }
```

`OnLoad` receives the logger, event registry, command registry, scheduler, and
data directory. Register callbacks there. `OnEnable` runs after loading and
`OnDisable` runs during orderly shutdown or failed enable cleanup.

Listeners and command callbacks run synchronously and serially for one plugin.
Do not block them. Scheduler callbacks run asynchronously. When a plugin is
disabled, GoCraft unregisters its listeners and commands and cancels its tasks.
Panics at every callback boundary are recovered and logged with a stack trace.

## Events

The first API version exposes:

| Event | Timing | Cancellable |
| --- | --- | --- |
| `PlayerJoinEvent` | after the player is reachable | no |
| `BlockBreakEvent` | before the block mutation | yes |

Java and Bedrock actions produce these same event types. No packet or numeric
protocol IDs are exposed. Cancelling `BlockBreakEvent` keeps the block intact
for either edition.

```go
ctx.Events().OnBlockBreak(func(event *gocraft.BlockBreakEvent) {
    if event.Block.ID == "minecraft:diamond_block" {
        event.Cancel()
    }
})
```

## Commands

Commands are declared in the bundle's generated `commands.pb`. Register the
matching executor ID during `OnLoad`:

```go
ctx.Commands().Register(1, func(call *gocraft.CommandContext) error {
    call.Reply("Hello, " + call.SenderName)
    return nil
})
```

The host parses and validates arguments before the callback. Typed values are
available through `CommandContext.Args`. Replies work for players and console
senders.

## Build a bundle

See [`examples/go-plugin`](../examples/go-plugin/) for a complete plugin. Build
its executable for the server operating system, place it at the manifest's
`entry`, then package it:

```sh
go run ./cmd/gocraft-cli build -o my-plugin.gcpkg ./my-plugin
```

Native binaries are operating-system and architecture specific. Hot reload and
in-process unloading are not supported. Rebuild plugins after a Plugin API
version change; the host rejects incompatible manifests before executing code.
