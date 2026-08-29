package ipc

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shortTempDir keeps the socket path inside the 107 byte budget. The default
// per-test temporary directory is named after the test, which is long enough to
// blow it on its own.
func shortTempDir(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp("", "gc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	return directory
}

func TestListenAcceptsAConnection(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "a.sock")
	listener, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the socket file was not created: %v", err)
	}
}

// Closing removes the file, so the ordinary path needs no cleanup of its own.
func TestClosingAListenerRemovesTheSocketFile(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "b.sock")
	listener, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the socket file outlived its listener: %v", err)
	}
}

// A host killed outright leaves its socket file behind, and the file alone
// blocks the next bind. Restarting must not require the admin to delete it.
func TestListenClearsAStaleSocketFile(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "c.sock")
	if err := os.WriteFile(path, []byte("left over"), 0o600); err != nil {
		t.Fatal(err)
	}
	listener, err := Listen(path)
	if err != nil {
		t.Fatalf("Listen() = %v, want a stale file to be cleared", err)
	}
	listener.Close()
}

// The dangerous half of the rule above: a path someone is actually serving must
// be refused, not taken over. Getting this wrong would let one server pull the
// socket out from under another.
func TestListenRefusesToStealALiveSocket(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "d.sock")
	live, err := Listen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()

	second, err := Listen(path)
	if err == nil {
		second.Close()
		t.Fatal("Listen() took over a socket that was already being served")
	}
	if !strings.Contains(err.Error(), "live process") {
		t.Fatalf("Listen() error = %v, want it to name the live process", err)
	}
	// The live listener must be untouched: its file still there, and still
	// accepting. A refusal that removed the file anyway would be worse than the
	// theft it was meant to prevent.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the live socket file was removed: %v", err)
	}
	client, err := net.Dial("unix", path)
	if err != nil {
		t.Fatalf("the live socket stopped accepting connections: %v", err)
	}
	client.Close()
}

func TestListenRejectsAnOverlongPath(t *testing.T) {
	directory := shortTempDir(t)
	path := filepath.Join(directory, strings.Repeat("x", maximumSocketPath)+".sock")
	_, err := Listen(path)
	if err == nil {
		t.Fatal("Listen() accepted a path over the limit")
	}
	// "bind: invalid argument" is what the kernel says; the caller needs better.
	if !strings.Contains(err.Error(), "107 byte limit") {
		t.Fatalf("Listen() error = %v, want it to name the limit", err)
	}
}

func TestSocketPathStaysInsideTheLimit(t *testing.T) {
	path, err := SocketPath(shortTempDir(t), "jvm")
	if err != nil {
		t.Fatal(err)
	}
	if len(path) > maximumSocketPath {
		t.Fatalf("SocketPath() = %q, %d bytes", path, len(path))
	}
	if !strings.Contains(filepath.Base(path), "jvm") {
		t.Fatalf("SocketPath() = %q, want the runtime name in it", path)
	}
}

func TestSocketPathRefusesADirectoryThatLeavesNoRoom(t *testing.T) {
	_, err := SocketPath(strings.Repeat("d", maximumSocketPath), "jvm")
	if err == nil {
		t.Fatal("SocketPath() accepted a directory with no room left")
	}
	if !strings.Contains(err.Error(), "shorter directory") {
		t.Fatalf("SocketPath() error = %v, want it to say what to do", err)
	}
}

// Two runtimes in one server must not collide, and the name is what separates
// them once the process id is shared.
func TestSocketPathSeparatesRuntimes(t *testing.T) {
	directory := shortTempDir(t)
	jvm, err := SocketPath(directory, "jvm")
	if err != nil {
		t.Fatal(err)
	}
	python, err := SocketPath(directory, "python")
	if err != nil {
		t.Fatal(err)
	}
	if jvm == python {
		t.Fatalf("both runtimes got %q", jvm)
	}
}