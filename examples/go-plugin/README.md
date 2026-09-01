# Native Go plugin example

This plugin logs joins and block breaks and provides `/greet`. Its callbacks
run in a separate process; Java and Bedrock actions use the same event API.

From the GoCraft repository root:

```sh
go run examples/go-plugin/generate.go
go build -o examples/go-plugin/bin/example-go ./examples/go-plugin
go run ./cmd/gocraft-cli build -o example-go.gcpkg ./examples/go-plugin
```

Copy `example-go.gcpkg` into the server's `plugins/` directory and restart the
server. GoCraft creates `plugins/example-go/` for configuration and plugin data.

The executable is platform-specific. Build the bundle on the same operating
system and architecture as the server. Cross-compilation also works, for
example:

```sh
GOOS=linux GOARCH=amd64 go build -o examples/go-plugin/bin/example-go ./examples/go-plugin
```

The command tree is generated as `commands.pb`. `main.go` registers its handler
against the path through that tree — `greet` — and never against the executor ID
the tree assigns it.

Native plugins currently cannot be hot-reloaded or unloaded independently of
their process. Rebuild the plugin whenever GoCraft's Go version or Plugin API
version changes.
