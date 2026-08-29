package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sourceManifest = `id = "fr.oreo.hello"
version = "0.1.0"
api = 1
runtime = "jvm"
entry = "fr.oreo.hello.HelloPlugin"

[subscribe]
events = ["block.break"]
perms = ["hello.notify"]
`

func writeSource(t *testing.T, manifest string) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "plugin.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return directory
}

func TestValidateDescribesAGoodManifest(t *testing.T) {
	stdout, stderr, code := runCLI("validate", writeSource(t, sourceManifest))
	if code != exitOK {
		t.Fatalf("validate = %d (%s), want %d", code, stderr, exitOK)
	}
	for _, expected := range []string{"fr.oreo.hello", "0.1.0", "jvm", "block.break", "hello.notify"} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("stdout is missing %q\n%s", expected, stdout)
		}
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestValidateReportsTheOffendingLine(t *testing.T) {
	directory := writeSource(t, `id = "fr.oreo.hello"
version = "0.1.0"
api = 1
runtime = "jvm"
description = "nope"
`)
	stdout, stderr, code := runCLI("validate", directory)
	if code != exitFailure {
		t.Fatalf("validate = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr, `plugin.toml:5: unknown field "description"`) {
		t.Fatalf("stderr = %q", stderr)
	}
	if !strings.Contains(stderr, directory) {
		t.Fatalf("stderr = %q, want it to name the directory", stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

// A key inside a table is reported by its full path, so "description" under
// [subscribe] cannot be confused with a top-level one.
func TestValidateQualifiesUnknownFieldsInsideATable(t *testing.T) {
	directory := writeSource(t, sourceManifest+"description = \"nope\"\n")
	_, stderr, code := runCLI("validate", directory)
	if code != exitFailure {
		t.Fatalf("validate = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr, `plugin.toml:10: unknown field "subscribe.description"`) {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestValidateFailsOnAMissingManifest(t *testing.T) {
	_, stderr, code := runCLI("validate", t.TempDir())
	if code != exitFailure {
		t.Fatalf("validate = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr, "plugin.toml") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestValidateRejectsAnUnsupportedAPI(t *testing.T) {
	directory := writeSource(t, strings.Replace(sourceManifest, "api = 1", "api = 2", 1))
	_, stderr, code := runCLI("validate", directory)
	if code != exitFailure {
		t.Fatalf("validate = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr, "API 2 is unsupported") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestValidateNeedsExactlyOneDirectory(t *testing.T) {
	for _, args := range [][]string{{"validate"}, {"validate", "a", "b"}} {
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
