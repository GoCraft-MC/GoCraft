package jvm

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withEmbeddedJar substitutes the embedded blob for the length of one test.
// The real one is empty in this build, and will not be until gocraft-java
// ships, so the extraction path is exercised with stand-in bytes.
func withEmbeddedJar(t *testing.T, contents []byte) {
	t.Helper()
	saved := embeddedJar
	embeddedJar = contents
	t.Cleanup(func() { embeddedJar = saved })
}

// The name is the whole point: a different binary carries different bytes and
// therefore asks for a different file, so an updated server can never pick up
// the jar the previous one left behind.
func TestExtractJarNamesTheFileAfterItsContents(t *testing.T) {
	directory := t.TempDir()
	jar := []byte("pretend this is a jar")

	path, err := extractJar(directory, jar)
	if err != nil {
		t.Fatalf("extractJar() = %v", err)
	}
	sum := sha256.Sum256(jar)
	want := filepath.Join(directory, fmt.Sprintf("gocraft-runtime-%x.jar", sum[:8]))
	if path != want {
		t.Fatalf("extractJar() = %q, want %q", path, want)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != string(jar) {
		t.Fatalf("extracted jar = %q", written)
	}
}

func TestExtractJarGivesDifferentContentsDifferentNames(t *testing.T) {
	directory := t.TempDir()
	first, err := extractJar(directory, []byte("version one"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := extractJar(directory, []byte("version two"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("extractJar() reused %q for different contents", first)
	}
}

// Every boot calls this. Rewriting 3 MB each time would be waste, and doing it
// while another server has the file open would be worse.
func TestExtractJarLeavesAnAlreadyExtractedFileAlone(t *testing.T) {
	directory := t.TempDir()
	jar := []byte("pretend this is a jar")

	path, err := extractJar(directory, jar)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	again, err := extractJar(directory, jar)
	if err != nil {
		t.Fatalf("extractJar() second call = %v", err)
	}
	if again != path {
		t.Fatalf("extractJar() = %q on the second call, want %q", again, path)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("extractJar() rewrote a jar that was already there")
	}
}

func TestExtractJarCreatesTheCacheDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "cache", "gocraft")
	if _, err := extractJar(directory, []byte("jar")); err != nil {
		t.Fatalf("extractJar() = %v", err)
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatalf("extractJar() did not create %s: %v", directory, err)
	}
}

// No download can fix a build that carries no jar, so the message has to say
// what to build instead rather than suggesting a retry.
func TestEnsureJarExplainsABuildWithoutAJar(t *testing.T) {
	withEmbeddedJar(t, nil)

	_, err := New(Config{ExtractDirectory: t.TempDir()}).ensureJar()
	if err == nil {
		t.Fatal("ensureJar() invented a jar this build does not carry")
	}
	if !strings.Contains(err.Error(), "gocraft_jvm_jar") {
		t.Fatalf("ensureJar() error = %v, want the build tag named", err)
	}
	if !strings.Contains(err.Error(), "no download can fix it") {
		t.Fatalf("ensureJar() error = %v, want it to rule out a retry", err)
	}
}

func TestEnsureJarExtractsTheEmbeddedOne(t *testing.T) {
	withEmbeddedJar(t, []byte("embedded jar bytes"))
	directory := t.TempDir()

	path, err := New(Config{ExtractDirectory: directory}).ensureJar()
	if err != nil {
		t.Fatalf("ensureJar() = %v", err)
	}
	if filepath.Dir(path) != directory {
		t.Fatalf("ensureJar() = %q, want it under %q", path, directory)
	}
}

// A jar the admin already put on disk is used where it is: extracting a copy
// would only add a second file that can go stale.
func TestEnsureJarUsesAConfiguredJarInPlace(t *testing.T) {
	directory := t.TempDir()
	configured := filepath.Join(directory, "gocraft-runtime.jar")
	if err := os.WriteFile(configured, []byte("local build"), 0o644); err != nil {
		t.Fatal(err)
	}
	withEmbeddedJar(t, []byte("embedded jar bytes"))

	path, err := New(Config{JarPath: configured, ExtractDirectory: directory}).ensureJar()
	if err != nil {
		t.Fatalf("ensureJar() = %v", err)
	}
	if path != configured {
		t.Fatalf("ensureJar() = %q, want the configured jar %q", path, configured)
	}
}

func TestEnsureJarReportsAMissingConfiguredJar(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.jar")

	_, err := New(Config{JarPath: missing}).ensureJar()
	if err == nil {
		t.Fatal("ensureJar() accepted a configured jar that is not there")
	}
	if !strings.Contains(err.Error(), "configured runtime jar") {
		t.Fatalf("ensureJar() error = %v", err)
	}
}
