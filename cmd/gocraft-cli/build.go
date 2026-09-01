package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

// zipEpoch keeps bundles byte-identical across machines and runs. The zip
// format stores a modification time per entry, and a source file's own
// timestamp would leak the checkout date into the artifact.
var zipEpoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

func buildCommand(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("o", "", "output path (default: <id>.gcpkg in the current directory)")
	commands := flags.String("commands", "",
		"command trees a compiler extracted, written into the bundle at the manifest's [commands] tree")
	flags.Usage = func() {
		fmt.Fprint(stderr, `Usage: gocraft-cli build [-o <file>.gcpkg] <dir>

Packs a plugin source directory into a bundle, then reopens the result with the
host's own loader. A bundle that builds is a bundle that loads.

With -commands, the trees an annotation processor extracted are encoded and
added at the path the manifest's [commands] tree names. Executor ids are minted
here: they belong to the tree, and nothing that ran earlier has to guess them.

Flags must come before the directory.
`)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return exitUsage
	}
	directory := flags.Arg(0)

	// Decode first so a broken manifest is reported before anything is written.
	manifest, err := readSourceManifest(directory)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	path := *output
	if path == "" {
		path = manifest.ID + ".gcpkg"
	}

	generated, err := generatedEntries(manifest, *commands)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailure
	}

	written, err := writeBundle(directory, path, generated)
	if err != nil {
		os.Remove(path)
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	// The proof that matters: the artifact is opened by the very code the
	// server runs at boot, so build-time and load-time validation cannot
	// disagree. This is also what catches a [commands] tree that names a file
	// the source directory does not contain.
	bundle, err := gcpkg.Open(path)
	if err != nil {
		os.Remove(path)
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	describeBundle(stdout, path, bundle.Manifest, written)
	return exitOK
}

// collectEntries lists the files to pack, as slash-separated paths relative to
// the source directory. Names beginning with a dot are skipped at every level:
// .git, .idea and editor droppings have no business inside a bundle.
func collectEntries(directory string) ([]string, error) {
	var names []string
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == directory {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", directory, err)
	}
	sort.Strings(names)
	return names, nil
}

// generatedEntries is what the build produces rather than copies.
//
// The manifest decides where it goes: the path is already declared there for
// the host to read, so naming it again on the command line would be a second
// place it lives.
func generatedEntries(manifest gcpkg.Manifest, commands string) (map[string][]byte, error) {
	if commands == "" {
		return nil, nil
	}
	if manifest.CommandTree == "" {
		return nil, fmt.Errorf("-commands was given and %s declares no [commands] tree",
			gcpkg.ManifestFileName)
	}
	encoded, err := readCommandTree(commands)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{manifest.CommandTree: encoded}, nil
}

func writeBundle(directory, path string, generated map[string][]byte) ([]string, error) {
	// Listed before the output file is created, so building into the source
	// directory cannot pack the bundle into itself.
	names, err := collectEntries(directory)
	if err != nil {
		return nil, err
	}
	for name := range generated {
		if slices.Contains(names, name) {
			return nil, fmt.Errorf("%s is both in the source directory and generated; "+
				"one of them would be lost", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	archive := zip.NewWriter(file)
	for _, name := range names {
		var err error
		if contents, ok := generated[name]; ok {
			err = packBytes(archive, name, contents)
		} else {
			err = packEntry(archive, directory, name)
		}
		if err != nil {
			archive.Close()
			file.Close()
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		file.Close()
		return nil, fmt.Errorf("write %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close %s: %w", path, err)
	}
	return names, nil
}

func packEntry(archive *zip.Writer, directory, name string) error {
	source, err := os.Open(filepath.Join(directory, filepath.FromSlash(name)))
	if err != nil {
		return err
	}
	defer source.Close()
	writer, err := archive.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: zipEpoch,
	})
	if err != nil {
		return fmt.Errorf("add %s: %w", name, err)
	}
	if _, err := io.Copy(writer, source); err != nil {
		return fmt.Errorf("add %s: %w", name, err)
	}
	return nil
}

// packBytes writes something the build produced, stamped with the same fixed
// epoch as every copied file so the bundle stays byte-identical across runs.
func packBytes(archive *zip.Writer, name string, contents []byte) error {
	writer, err := archive.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: zipEpoch,
	})
	if err != nil {
		return fmt.Errorf("add %s: %w", name, err)
	}
	if _, err := writer.Write(contents); err != nil {
		return fmt.Errorf("add %s: %w", name, err)
	}
	return nil
}

func describeBundle(w io.Writer, path string, manifest gcpkg.Manifest, names []string) {
	size := int64(-1)
	if info, err := os.Stat(path); err == nil {
		size = info.Size()
	}
	fmt.Fprintf(w, "%s: built\n", path)
	fmt.Fprintf(w, "  id       %s %s\n", manifest.ID, manifest.Version)
	fmt.Fprintf(w, "  runtime  %s (api %d)\n", manifest.Runtime, manifest.APIVersion)
	fmt.Fprintf(w, "  entries  %d\n", len(names))
	for _, name := range names {
		fmt.Fprintf(w, "    %s\n", name)
	}
	if size >= 0 {
		fmt.Fprintf(w, "  size     %d bytes\n", size)
	}
}
