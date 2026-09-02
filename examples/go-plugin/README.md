# Native Go plugin example

This plugin logs joins and block breaks and provides `/greet`. Its callbacks
run in a separate process; Java and Bedrock actions use the same event API.

This directory is its own Go module, and that is deliberate: it requires a
published `gocraft-api-go` rather than the copy next door, so building it proves
what a plugin author's build does. From here:

```sh
go run . -gocraft-dump-commands .gocraft/commands.json
go build -o bin/example-go .
```

The first line asks the plugin what commands it has. It declares them once, in
`Commands()`, and that same declaration is what the loader binds handlers from —
so the shape in the bundle and the functions that answer cannot disagree. The
dump lands in a dot directory because the packer skips those.

Then package it. The build tool is its own module, so it needs no checkout of
anything:

```sh
go run github.com/GoCraft-MC/gocraft-cli@latest \
    build -commands .gocraft/commands.json -o example-go.gcpkg .
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

`gocraft-cli` turns that neutral file into the `commands.pb` the bundle ships —
the same program that does it for a Java plugin, from the same kind of file its
annotation processor writes. Executor ids are minted there and nowhere else,
which is why handlers bind to paths rather than to numbers.

Native plugins currently cannot be hot-reloaded or unloaded independently of
their process. Rebuild the plugin whenever GoCraft's Go version or Plugin API
version changes.
