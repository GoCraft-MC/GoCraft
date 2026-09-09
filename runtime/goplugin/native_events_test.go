package goplugin

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/plugin"
	"GoCraft/java/handler"
	"GoCraft/runtime/link"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

// Reuses the existing real process/socket harness, without server credentials.
func nativeEventRuntime(t testing.TB) (*Runtime, plugin.Bundle, *plugin.Bus) {
	t.Helper()
	sockets, err := os.MkdirTemp("", "gc-events-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(sockets) })
	runtime := New(Config{ExtractDirectory: t.TempDir(), SocketDirectory: sockets, StartTimeout: helperStartTimeout,
		Spawn: func(entry string) link.Spawn {
			return func(socket string) *exec.Cmd {
				command := helperSpawn(entry)(socket)
				command.Env = append(command.Env, "GOCRAFT_NATIVE_EVENTS=1")
				return command
			}
		},
	})
	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { runtime.Stop(context.Background()) })
	bundle := plugin.Bundle{Bundle: gcpkg.Bundle{
		Path: writeTestBundleWith(t, "bin/example", []byte("placeholder"), helperCommandTree(t)),
		Manifest: gcpkg.Manifest{ID: "example", Version: "1.0.0", APIVersion: 1, Runtime: RuntimeName, Entry: "bin/example", CommandTree: commandTreeEntry,
			Subscriptions: []gcpkg.Subscription{{Event: plugin.EventBlockBreak}, {Event: plugin.EventPlayerChat}}},
	}, DataDirectory: t.TempDir()}
	bus := plugin.NewBus(t.Context(), 2*time.Millisecond)
	return runtime, bundle, bus
}

func TestGameplayChatRoundTripThroughNativeProcessAndReload(t *testing.T) {
	runtime, bundle, bus := nativeEventRuntime(t)
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	d := handler.NewDispatcher()
	d.SetEventBus(bus)
	for range 2 {
		loaded, err := runtime.Load(t.Context(), bundle)
		if err != nil {
			t.Fatal(err)
		}
		if err := bus.Attach(loaded); err != nil {
			t.Fatal(err)
		}
		message := "original"
		if !d.FilterPlayerChat(p, &message) || message != "rewritten" {
			t.Fatal("IPC mutation lost")
		}
		message = "cancel"
		if d.FilterPlayerChat(p, &message) {
			t.Fatal("IPC cancellation lost")
		}
		bus.Detach("example")
		if err := loaded.Unload(t.Context()); err != nil {
			t.Fatal(err)
		}
		message = "original"
		if !d.FilterPlayerChat(p, &message) || message != "original" {
			t.Fatal("unloaded handler still ran")
		}
	}
}
