package jvm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"GoCraft/core/command"
	"GoCraft/core/plugin"
)

func TestNameIsWhatAManifestWrites(t *testing.T) {
	if New(Config{}).Name() != "jvm" {
		t.Fatalf("Name() = %q", New(Config{}).Name())
	}
}

// Provision resolves the JDK and Start spawns it. Reversing them would spawn
// nothing, and saying so beats an exec error naming an empty path.
func TestStartRefusesBeforeProvision(t *testing.T) {
	err := New(Config{}).Start(t.Context(), nil)
	if err == nil {
		t.Fatal("Start() ran without a provisioned JDK")
	}
	if !strings.Contains(err.Error(), "Provision") {
		t.Fatalf("Start() error = %v, want the missing step named", err)
	}
}

func TestWorkBeforeStartIsRefused(t *testing.T) {
	runtime := New(Config{})
	for name, call := range map[string]func() error{
		"Load": func() error {
			_, err := runtime.Load(t.Context(), plugin.Bundle{
				Manifest: plugin.Manifest{ID: "dev.example.shop"},
			})
			return err
		},
		"Ready": func() error { return runtime.Ready(t.Context()) },
	} {
		err := call()
		if err == nil {
			t.Fatalf("%s() succeeded with no JVM running", name)
		}
		if !strings.Contains(err.Error(), "not running") {
			t.Fatalf("%s() error = %v", name, err)
		}
	}
	if runtime.Failed() != nil {
		t.Fatal("Failed() returned a channel before Start")
	}
}

// Stop on a runtime that never started is how a rolled-back boot unwinds.
func TestStopBeforeStartIsNotAnError(t *testing.T) {
	if err := New(Config{}).Stop(t.Context()); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
}

// The envelope reserves the fields for command invocation and never defines
// them, so nothing can reach a plugin's executor across the process boundary.
// Loading anyway would hand an admin a plugin whose commands silently do
// nothing, which is the worst of the three possible outcomes.
func TestLoadRefusesABundleWithCommands(t *testing.T) {
	_, err := New(Config{}).Load(t.Context(), plugin.Bundle{
		Manifest: plugin.Manifest{ID: "dev.example.shop"},
		Commands: &command.Root{},
	})
	if err == nil {
		t.Fatal("Load() accepted a command tree the ABI cannot carry")
	}
	// The bundle is judged before the process is: the answer is the same
	// whether the JVM is up or not, and naming the real reason beats telling an
	// admin the runtime is merely not running.
	if !strings.Contains(err.Error(), "commands") {
		t.Fatalf("Load() error = %v, want commands named as the reason", err)
	}
}

func TestSpawnBuildsTheDocumentedCommandLine(t *testing.T) {
	runtime := New(Config{})
	spawned := runtime.spawn(filepath.FromSlash("/opt/jdk25/bin/java"), filepath.FromSlash("/cache/rt.jar"))(
		filepath.FromSlash("/tmp/gc-jvm-1.sock"))

	if spawned.Path != filepath.FromSlash("/opt/jdk25/bin/java") {
		t.Fatalf("spawn() path = %q", spawned.Path)
	}
	want := []string{
		filepath.FromSlash("/opt/jdk25/bin/java"),
		// Not decoration: without it the JVM prints four sun.misc.Unsafe
		// deprecation warnings, from protobuf, on every boot.
		"--sun-misc-unsafe-memory-access=allow",
		"-jar", filepath.FromSlash("/cache/rt.jar"),
		"--sock", filepath.FromSlash("/tmp/gc-jvm-1.sock"),
		"--abi", "1",
	}
	if len(spawned.Args) != len(want) {
		t.Fatalf("spawn() args = %v, want %v", spawned.Args, want)
	}
	for index, argument := range want {
		if spawned.Args[index] != argument {
			t.Fatalf("spawn() args[%d] = %q, want %q", index, spawned.Args[index], argument)
		}
	}
	// Where a runtime's own logs go is its business, and by default that is the
	// server console, which is already teed into latest.log.
	if spawned.Stdout != os.Stdout || spawned.Stderr != os.Stderr {
		t.Fatal("spawn() left the JVM's output unrouted")
	}
}

func TestSpawnHonoursConfiguredOutput(t *testing.T) {
	var out, errs strings.Builder
	spawned := New(Config{Stdout: &out, Stderr: &errs}).spawn("java", "rt.jar")("s.sock")

	if spawned.Stdout != &out || spawned.Stderr != &errs {
		t.Fatal("spawn() ignored the configured writers")
	}
}

func TestSocketDirectoryFallsBackToTemp(t *testing.T) {
	if got := New(Config{}).socketDirectory(); got != os.TempDir() {
		t.Fatalf("socketDirectory() = %q, want the temporary directory", got)
	}
	if got := New(Config{SocketDirectory: "sock"}).socketDirectory(); got != "sock" {
		t.Fatalf("socketDirectory() = %q, want the configured one", got)
	}
}

// The assertions that matter most in this package: the host drives Java through
// these interfaces and nothing else, so a signature drifting out of line has to
// fail here rather than at the call site in core/plugin.
var (
	_ plugin.Runtime      = (*Runtime)(nil)
	_ plugin.ReadyRuntime = (*Runtime)(nil)
	_ plugin.Instance     = (*Instance)(nil)
)

// A Java plugin cannot answer a command yet, so the runtime must not claim it
// can: core/plugin refuses a bundle whose instance does not implement this, and
// a false positive there would surface as a command that does nothing.
func TestInstanceDoesNotClaimCommandSupport(t *testing.T) {
	var instance plugin.Instance = &Instance{}
	if _, ok := instance.(plugin.CommandInstance); ok {
		t.Fatal("Instance claims command support the ABI cannot deliver")
	}
}

func TestInstanceReportsItsManifest(t *testing.T) {
	manifest := plugin.Manifest{ID: "dev.example.shop", Version: "1.2.0", Runtime: "jvm"}
	instance := &Instance{manifest: manifest}
	if instance.Manifest().ID != manifest.ID {
		t.Fatalf("Manifest() = %+v", instance.Manifest())
	}
}
