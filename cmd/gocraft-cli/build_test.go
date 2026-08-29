package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildInto(t *testing.T, source string) (path string, stdout, stderr string, code int) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "out.gcpkg")
	stdout, stderr, code = runCLI("build", "-o", path, source)
	return path, stdout, stderr, code
}

func archiveNames(t *testing.T, path string) []string {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	names := make([]string, 0, len(archive.File))
	for _, file := range archive.File {
		names = append(names, file.Name)
	}
	return names
}

func TestBuildProducesALoadableBundle(t *testing.T) {
	path, stdout, stderr, code := buildInto(t, writeSource(t, sourceManifest))
	if code != exitOK {
		t.Fatalf("build = %d (%s), want %d", code, stderr, exitOK)
	}
	if names := archiveNames(t, path); len(names) != 1 || names[0] != "plugin.toml" {
		t.Fatalf("archive entries = %v, want [plugin.toml]", names)
	}
	if !strings.Contains(stdout, "fr.oreo.hello") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestBuildIsReproducible(t *testing.T) {
	source := writeSource(t, sourceManifest)
	first, _, stderr, code := buildInto(t, source)
	if code != exitOK {
		t.Fatalf("build = %d (%s)", code, stderr)
	}
	second, _, _, _ := buildInto(t, source)
	left, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatal("two builds of the same source produced different bytes")
	}
}

func TestBuildSkipsDotEntries(t *testing.T) {
	source := writeSource(t, sourceManifest)
	if err := os.WriteFile(filepath.Join(source, ".DS_Store"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, ".git", "HEAD"), []byte("ref: x"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, _, stderr, code := buildInto(t, source)
	if code != exitOK {
		t.Fatalf("build = %d (%s)", code, stderr)
	}
	for _, name := range archiveNames(t, path) {
		if strings.HasPrefix(name, ".") || strings.Contains(name, "/.") {
			t.Fatalf("archive contains a dot entry: %s", name)
		}
	}
}

func TestBuildKeepsNestedFilesWithSlashPaths(t *testing.T) {
	source := writeSource(t, sourceManifest)
	if err := os.MkdirAll(filepath.Join(source, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "lib", "extra.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, _, stderr, code := buildInto(t, source)
	if code != exitOK {
		t.Fatalf("build = %d (%s)", code, stderr)
	}
	names := archiveNames(t, path)
	found := false
	for _, name := range names {
		if name == "lib/extra.jar" {
			found = true
		}
		if strings.Contains(name, `\`) {
			t.Fatalf("archive entry uses a backslash: %q", name)
		}
	}
	if !found {
		t.Fatalf("archive entries = %v, want lib/extra.jar", names)
	}
}

// The post-build reopen is what catches this: the manifest is valid on its own,
// but the file it points at is not in the source directory.
func TestBuildRejectsAMissingCommandTree(t *testing.T) {
	source := writeSource(t, sourceManifest+"\n[commands]\ntree = \"commands.pb\"\n")
	path, stdout, stderr, code := buildInto(t, source)
	if code != exitFailure {
		t.Fatalf("build = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr, "commands.pb") {
		t.Fatalf("stderr = %q", stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a rejected build left its output behind")
	}
}

func TestBuildRejectsABadManifestWithoutWriting(t *testing.T) {
	source := writeSource(t, sourceManifest+"description = \"nope\"\n")
	path, _, stderr, code := buildInto(t, source)
	if code != exitFailure {
		t.Fatalf("build = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr, "unknown field") {
		t.Fatalf("stderr = %q", stderr)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a rejected build left its output behind")
	}
}

func TestBuildNeedsExactlyOneDirectory(t *testing.T) {
	for _, args := range [][]string{{"build"}, {"build", "a", "b"}} {
		stdout, stderr, code := runCLI(args...)
		if code != exitUsage {
			t.Fatalf("run(%v) = %d, want %d", args, code, exitUsage)
		}
		if !strings.Contains(stderr, "Usage:") {
			t.Fatalf("run(%v) stderr = %q", args, stderr)
		}
		if stdout != "" {
			t.Fatalf("run(%v) stdout = %q, want empty", args, stdout)
		}
	}
}
