package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(args ...string) (stdout, stderr string, code int) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return out.String(), errOut.String(), code
}

func TestVersionGoesToStdout(t *testing.T) {
	stdout, stderr, code := runCLI("version")
	if code != exitOK {
		t.Fatalf("run(version) = %d, want %d", code, exitOK)
	}
	if !strings.HasPrefix(stdout, "gocraft-cli ") {
		t.Fatalf("stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	for _, argument := range []string{"help", "-h", "--help"} {
		stdout, stderr, code := runCLI(argument)
		if code != exitOK {
			t.Fatalf("run(%s) = %d, want %d", argument, code, exitOK)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Fatalf("run(%s) stdout = %q", argument, stdout)
		}
		if stderr != "" {
			t.Fatalf("run(%s) stderr = %q, want empty", argument, stderr)
		}
	}
}

func TestNoArgumentsIsAUsageError(t *testing.T) {
	stdout, stderr, code := runCLI()
	if code != exitUsage {
		t.Fatalf("run() = %d, want %d", code, exitUsage)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestUnknownCommandNamesItOnStderr(t *testing.T) {
	stdout, stderr, code := runCLI("frobnicate")
	if code != exitUsage {
		t.Fatalf("run(frobnicate) = %d, want %d", code, exitUsage)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, `unknown command "frobnicate"`) {
		t.Fatalf("stderr = %q", stderr)
	}
}