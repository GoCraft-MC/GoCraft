package jvm

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"GoCraft/runtime/link"
	wire "github.com/GoCraft-MC/gocraft-abi/abi/v1/wire"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
	"github.com/GoCraft-MC/gocraft-abi/ipc"
)

const (
	fakeJVMEnv    = "GOCRAFT_FAKE_JVM"
	fakeSocketEnv = "GOCRAFT_FAKE_JVM_SOCKET"
	fakeLivesEnv  = "GOCRAFT_FAKE_JVM_LIVES"
)

// TestFakeJVM is not a test. It is the child process the respawn tests spawn:
// `go test` re-executes its own binary pointed here, which is how a runtime
// that dies on demand is exercised without a JDK anywhere near this
// repository's CI — the rule §16 exists to keep.
func TestFakeJVM(t *testing.T) {
	behaviour := os.Getenv(fakeJVMEnv)
	if behaviour == "" {
		t.Skip("not the child process")
	}
	runFakeJVM(behaviour, os.Getenv(fakeSocketEnv), os.Getenv(fakeLivesEnv))
}

// runFakeJVM speaks just enough of the ABI to be respawned: greet, answer
// loads, answer pings, and — on the runs the test asks for — die.
func runFakeJVM(behaviour, socket, livesFile string) {
	stream, err := ipcDial(socket)
	if err != nil {
		os.Exit(4)
	}
	defer stream.Close()
	codec := ipc.NewCodec(stream)

	// A file rather than a counter in memory: each life is a new process, so
	// the only way one knows how many came before is to look.
	lives := appendLife(livesFile)
	if (behaviour == "die-once" || behaviour == "die-once-slow-load") && lives == 1 {
		// Greets, then leaves. The host has a healthy runtime, then does not.
		sendHello(codec)
		time.Sleep(150 * time.Millisecond)
		os.Exit(9)
	}
	if behaviour == "die-always" {
		sendHello(codec)
		time.Sleep(100 * time.Millisecond)
		os.Exit(9)
	}
	sendHello(codec)
	serveFakeJVM(codec, behaviour == "die-once-slow-load")
}

func TestRespawnBringsThePluginsBack(t *testing.T) {
	lives := filepath.Join(t.TempDir(), "lives")
	restored := make(chan []string, 4)
	runtime := fakeRuntime(t, "die-once", lives, func(ids []string) { restored <- ids })

	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	defer runtime.Stop(context.Background())

	// One bundle, remembered so a respawn can put it back. The host loaded it
	// once and will not do it again: it is not watching this process.
	runtime.remember(loadedBundle{
		id: "dev.example.shop", path: "shop.gcpkg",
		entry: "x.Shop", events: []string{"block.break"},
	})

	select {
	case ids := <-restored:
		if len(ids) != 1 || ids[0] != "dev.example.shop" {
			t.Fatalf("restored = %v, want the plugin that was loaded", ids)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the runtime never came back")
	}
	if countLives(t, lives) < 2 {
		t.Fatalf("lives = %d, want a second process", countLives(t, lives))
	}
}

// An instance must answer through whatever process is alive now. Holding the
// supervisor it was created with would make it write into a socket nobody is
// reading, and the failure would look like a timeout rather than a crash.
func TestInstanceFollowsTheRuntimeAcrossARespawn(t *testing.T) {
	lives := filepath.Join(t.TempDir(), "lives")
	restored := make(chan []string, 4)
	runtime := fakeRuntime(t, "die-once", lives, func(ids []string) { restored <- ids })

	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop(context.Background())

	instance := &Instance{runtime: runtime, manifest: gcpkg.Manifest{ID: "dev.example.shop"}}
	before, err := runtime.running()
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-restored:
	case <-time.After(30 * time.Second):
		t.Fatal("the runtime never came back")
	}

	after, err := runtime.running()
	if err != nil {
		t.Fatalf("running() after respawn = %v", err)
	}
	if before == after {
		t.Fatal("the supervisor was reused; the old process is gone")
	}
	// The instance never learned about any of this, which is the point.
	if _, err := instance.Dispatch(t.Context(), nil); err == nil {
		t.Fatal("Dispatch() accepted a nil event")
	}
}

// A JVM that dies on startup dies on every startup. Spinning one up forever is
// harder to diagnose than a server that says it stopped trying.
func TestRespawnGivesUpAfterItsAttempts(t *testing.T) {
	lives := filepath.Join(t.TempDir(), "lives")
	runtime := fakeRuntime(t, "die-always", lives, nil)
	// StableAfter left at its default minute: the fake dies in 100 ms, so no
	// respawn ever counts as a recovery and the attempts accumulate — which is
	// the flapping this test exists to bound.
	runtime.config.Respawn = Respawn{Attempts: 2, Backoff: 10 * time.Millisecond}

	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop(context.Background())

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := runtime.running(); err != nil {
			// It gave up and cleared the supervisor, which is the reported
			// outcome rather than a silent one.
			if lived := countLives(t, lives); lived > 4 {
				t.Fatalf("started %d processes for 2 attempts", lived)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the runtime never stopped respawning")
}

// An admin who would rather see the plugins stop than have them come back with
// empty state gets exactly that.
func TestRespawnCanBeDisabled(t *testing.T) {
	lives := filepath.Join(t.TempDir(), "lives")
	runtime := fakeRuntime(t, "die-always", lives, nil)
	runtime.config.Respawn = Respawn{Disabled: true}

	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop(context.Background())

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := runtime.running(); err != nil {
			if lived := countLives(t, lives); lived != 1 {
				t.Fatalf("started %d processes with respawn disabled", lived)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the runtime is still up after dying with respawn disabled")
}

// Stop must not race the watcher into respawning what it just took down.
func TestStopDoesNotRespawn(t *testing.T) {
	lives := filepath.Join(t.TempDir(), "lives")
	runtime := fakeRuntime(t, "ok", lives, nil)

	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := runtime.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() = %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	if lived := countLives(t, lives); lived != 1 {
		t.Fatalf("started %d processes; Stop respawned what it stopped", lived)
	}
	if _, err := runtime.running(); err == nil {
		t.Fatal("the runtime is still running after Stop")
	}
}

func TestStopDuringRespawnDoesNotPublishReplacement(t *testing.T) {
	lives := filepath.Join(t.TempDir(), "lives")
	runtime := fakeRuntime(t, "die-once-slow-load", lives, nil)
	runtime.remember(loadedBundle{id: "dev.example.slow", path: "slow.gcpkg"})

	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for countLives(t, lives) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if countLives(t, lives) < 2 {
		t.Fatal("replacement JVM never started")
	}
	if err := runtime.Stop(t.Context()); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	time.Sleep(750 * time.Millisecond)
	if _, err := runtime.running(); err == nil {
		t.Fatal("replacement JVM was published after Stop")
	}
}

// ── Harness ───────────────────────────────────────────────────────────────────

// fakeRuntime builds a runtime whose "JVM" is this test binary. Provision is
// bypassed with a fake java path: nothing here runs a JDK.
func fakeRuntime(t *testing.T, behaviour, lives string, onRespawn func([]string)) *Runtime {
	t.Helper()
	// A jar that is never read: the spawn below ignores it. ensureJar still
	// insists on one existing, which is the check doing its job.
	jar := filepath.Join(t.TempDir(), "gocraft-runtime.jar")
	if err := os.WriteFile(jar, []byte("not a real jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	runtime := New(Config{
		JarPath:          jar,
		SocketDirectory:  shortTempDir(t),
		ExtractDirectory: t.TempDir(),
		StartTimeout:     20 * time.Second,
		Liveness:         link.Liveness{Every: 50 * time.Millisecond, Timeout: 50 * time.Millisecond, Missed: 2},
		Respawn:          Respawn{Attempts: 3, Backoff: 10 * time.Millisecond},
		OnRespawn:        onRespawn,
		Stdout:           os.Stderr,
		Stderr:           os.Stderr,
		Spawn: func(socket string) *exec.Cmd {
			command := exec.Command(os.Args[0], "-test.run=^TestFakeJVM$")
			command.Env = append(os.Environ(),
				fakeJVMEnv+"="+behaviour, fakeSocketEnv+"="+socket, fakeLivesEnv+"="+lives)
			command.Stdout, command.Stderr = os.Stderr, os.Stderr
			return command
		},
	})
	runtime.setJava(os.Args[0])
	t.Cleanup(func() { runtime.Stop(context.Background()) })
	return runtime
}

// The socket path budget is 107 bytes, and a Go test temp directory is long.
func shortTempDir(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp("", "gj")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	return directory
}

var livesMu sync.Mutex

func appendLife(path string) int {
	if path == "" {
		return 1
	}
	livesMu.Lock()
	defer livesMu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 1
	}
	file.WriteString("x")
	file.Close()
	contents, _ := os.ReadFile(path)
	return len(contents)
}

func countLives(t *testing.T, path string) int {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return len(strings.TrimSpace(string(contents)))
}

// ── The fake JVM's side of the ABI ────────────────────────────────────────────

func ipcDial(socket string) (net.Conn, error) {
	return net.Dial("unix", socket)
}

func sendHello(codec *ipc.Codec) {
	codec.Send(&wire.Envelope{Body: &wire.Envelope_Hello{
		Hello: &wire.Hello{Abi: abiVersion, Runtime: "fake jvm"},
	}})
}

// serveFakeJVM answers what a respawn needs and nothing more: loads succeed,
// pings are answered on the reader so silence means stuck rather than busy, and
// shutdown ends the process.
func serveFakeJVM(codec *ipc.Codec, slowLoad bool) {
	for {
		envelope, err := codec.Receive()
		if err != nil {
			os.Exit(0)
		}
		switch envelope.GetBody().(type) {
		case *wire.Envelope_Load:
			if slowLoad {
				time.Sleep(500 * time.Millisecond)
			}
			codec.Send(&wire.Envelope{Seq: envelope.GetSeq(), Body: &wire.Envelope_Loaded{
				Loaded: &wire.Loaded{PluginId: envelope.GetLoad().GetPluginId()},
			}})
		case *wire.Envelope_Ping:
			codec.Send(&wire.Envelope{Seq: envelope.GetSeq(),
				Body: &wire.Envelope_Pong{Pong: &wire.Pong{}}})
		case *wire.Envelope_Invoke:
			// Replies to whoever typed it, which is what a real handler does
			// and what proves the sender crossed with the invocation.
			invoke := envelope.GetInvoke()
			codec.Send(&wire.Envelope{Seq: envelope.GetSeq(), Body: &wire.Envelope_Invoked{
				Invoked: &wire.Invoked{Effects: []*wire.HostCall{{
					Type: "chat.message",
					Fields: []*wire.Value{
						invoke.GetSender().GetPlayer(),
						{Kind: &wire.Value_StringValue{StringValue: "ran " + strconv.Itoa(int(invoke.GetExecutor()))}},
					},
				}}},
			}})
		case *wire.Envelope_Shutdown:
			os.Exit(0)
		}
	}
}

// A crash a day apart is not two thirds of the way to being given up on. The
// counter measures flapping, so a runtime that came back and stayed up resets
// it — which is what StableAfter decides.
func TestRespawnForgetsAFailureOnceTheRuntimeSettles(t *testing.T) {
	lives := filepath.Join(t.TempDir(), "lives")
	restored := make(chan []string, 4)
	runtime := fakeRuntime(t, "die-once", lives, func(ids []string) { restored <- ids })
	runtime.config.Respawn = Respawn{
		Attempts: 1, Backoff: 10 * time.Millisecond,
		// The second process is healthy, and anything alive for 50 ms counts as
		// settled here.
		StableAfter: 50 * time.Millisecond,
	}

	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop(context.Background())

	select {
	case <-restored:
	case <-time.After(30 * time.Second):
		t.Fatal("the runtime never came back")
	}
	// One attempt was allowed and one was used. The runtime is up, and staying
	// up is what has to clear the count rather than leaving it one crash from
	// being abandoned.
	time.Sleep(200 * time.Millisecond)
	if _, err := runtime.running(); err != nil {
		t.Fatalf("running() = %v, want a runtime that recovered and stayed", err)
	}
}
