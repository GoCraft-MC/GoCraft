package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"GoCraft/core/plugin"
)

// validateCommand checks a plugin source directory without producing anything.
// It comes before build on purpose: these messages are the first thing a plugin
// author reads, and until now the only reader of a manifest was the server, at
// boot, with a refused startup as its whole explanation.
func validateCommand(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprint(stderr, `Usage: gocraft-cli validate <dir>

Checks the plugin.toml at the root of a plugin source directory, using the
same decoder the server uses when it opens a built bundle.
`)
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return exitUsage
	}
	directory := flags.Arg(0)
	manifest, err := readSourceManifest(directory)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	describeManifest(stdout, directory, manifest)
	return exitOK
}

// readSourceManifest decodes the manifest of an unpacked plugin directory. It
// calls the very decoder the host runs against a packed bundle, so a manifest
// this accepts is a manifest the server accepts.
func readSourceManifest(directory string) (plugin.Manifest, error) {
	file, err := os.Open(filepath.Join(directory, plugin.ManifestFileName))
	if err != nil {
		return plugin.Manifest{}, err
	}
	defer file.Close()
	manifest, err := plugin.DecodeManifest(file)
	if err != nil {
		// The decoder already reports plugin.toml and a line; only the
		// directory is missing, and it is what disambiguates several plugins.
		return plugin.Manifest{}, fmt.Errorf("%s: %w", directory, err)
	}
	return manifest, nil
}

func describeManifest(w io.Writer, directory string, manifest plugin.Manifest) {
	fmt.Fprintf(w, "%s: ok\n", directory)
	fmt.Fprintf(w, "  id       %s\n", manifest.ID)
	fmt.Fprintf(w, "  version  %s\n", manifest.Version)
	fmt.Fprintf(w, "  runtime  %s (api %d)\n", manifest.Runtime, manifest.APIVersion)
	if manifest.Entry != "" {
		fmt.Fprintf(w, "  entry    %s\n", manifest.Entry)
	}
	if manifest.CommandTree != "" {
		fmt.Fprintf(w, "  commands %s\n", manifest.CommandTree)
	}
	if len(manifest.Subscriptions) > 0 {
		events := make([]string, 0, len(manifest.Subscriptions))
		for _, subscription := range manifest.Subscriptions {
			events = append(events, subscription.Event)
		}
		fmt.Fprintf(w, "  events   %s\n", strings.Join(events, ", "))
	}
	if len(manifest.Permissions) > 0 {
		fmt.Fprintf(w, "  perms    %s\n", strings.Join(manifest.Permissions, ", "))
	}
}
