# Native Go plugin example

This plugin logs joins and block breaks and provides `/greet`. Its callbacks
run in a separate process; Java and Bedrock actions use the same event API.

This directory is its own Go module, and that is deliberate: it requires a
published `gocraft-api-go` rather than the copy next door, so building it proves
what a plugin author's build does. From here:

```sh
go run generate.go
go build -o bin/example-go .
```

Then package it. The build tool is its own module, so it needs no checkout of
anything:

```sh
go run github.com/GoCraft-MC/gocraft-cli@latest build -o example-go.gcpkg .
```

It reads the directory, it does not compile it.

Copy `example-go.gcpkg` into the server's `plugins/` directory and restart the
server. GoCraft creates `plugins/example-go/` for configuration and plugin data.

The executable is platform-specific. Build the bundle on the same operating
system and architecture as the server. Cross-compilation also works, for
example:

```sh
GOOS=linux GOARCH=amd64 go build -o bin/example-go .
```

The command tree is generated as `commands.pb`. `main.go` registers its handler
against the path through that tree — `greet` — and never against the executor ID
the tree assigns it.

Native plugins currently cannot be hot-reloaded or unloaded independently of
their process. Rebuild the plugin whenever GoCraft's Go version or Plugin API
version changes.
