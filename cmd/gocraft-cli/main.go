// Command gocraft-cli builds and validates GoCraft plugin bundles.
//
// It is a separate binary from the server on purpose: a plugin author compiles,
// they never run a server. Nothing under cmd/gocraft-cli may import server/,
// java/ or bedrock/ — the linker only keeps what main can reach, so that rule
// alone is what keeps the tool small.
//
// The bundle format and its validation are not reimplemented here. They live in
// core/plugin, which the server also uses, so a bundle that builds is a bundle
// that loads.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is overridden at build time via -ldflags, like the server binary.
var version = "dev"

// Exit codes are distinct so a build script can tell a bad invocation from a
// plugin that genuinely failed.
const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run holds the dispatch so commands can be exercised without spawning a
// process, and so every write goes through an injected writer rather than
// straight to the standard streams.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	switch args[0] {
	case "validate":
		return validateCommand(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, "gocraft-cli", version)
		return exitOK
	case "help", "-h", "--help":
		usage(stdout)
		return exitOK
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		usage(stderr)
		return exitUsage
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `gocraft-cli - build and validate GoCraft plugin bundles

Usage:
  gocraft-cli <command> [arguments]

Commands:
  validate <dir>   Check the plugin.toml of a plugin source directory
  version          Print the tool version
  help             Print this message
`)
}