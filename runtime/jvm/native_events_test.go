package jvm

import (
	"context"
	"os"
	"testing"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

// Optional real-JVM integration: the existing Gradle test builds this fixture.
// No credentials, downloaded plugins or production server are required.
func nativeJVMFixture(t testing.TB) (*plugin.Bus, *Instance) {
	t.Helper()
	jar, fixture := os.Getenv("GOCRAFT_EVENT_RUNTIME_JAR"), os.Getenv("GOCRAFT_EVENT_FIXTURE")
	if jar == "" || fixture == "" {
		t.Skip("set GOCRAFT_EVENT_RUNTIME_JAR and GOCRAFT_EVENT_FIXTURE after the Gradle build")
	}
	sockets, err := os.MkdirTemp("", "gc-jvm-events-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(sockets) })
	runtime := New(Config{PreferSystem: true, JarPath: jar, SocketDirectory: sockets, ExtractDirectory: t.TempDir(), StartTimeout: 10 * time.Second, EventBudget: 2 * time.Millisecond})
	if err := runtime.Provision(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { runtime.Stop(context.Background()) })
	loaded, err := runtime.Load(t.Context(), plugin.Bundle{Bundle: gcpkg.Bundle{Path: fixture, Manifest: gcpkg.Manifest{
		ID: "test.chat", Version: "1.0.0", APIVersion: 1, Runtime: RuntimeName, Entry: "test.plugin.BenchmarkPlugin",
		Subscriptions: []gcpkg.Subscription{{Event: plugin.EventPlayerChat}},
	}}, DataDirectory: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("cold JVM start and plugin load: %s", time.Since(started))
	bus := plugin.NewBus(t.Context(), 2*time.Millisecond)
	if err := bus.Attach(loaded); err != nil {
		t.Fatal(err)
	}
	return bus, loaded.(*Instance)
}

func TestNativeEventRoundTripThroughRealJVM(t *testing.T) {
	bus, loaded := nativeJVMFixture(t)
	event := &abi.Event{Type: plugin.EventPlayerChat, Fields: plugin.BlankEvent(plugin.EventPlayerChat)}
	if err := loaded.Warm(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	message := "original"
	if !bus.EmitPlayerChat(p, &message) || message != "rewritten" {
		t.Fatal("Java mutation lost through IPC")
	}
	message = "cancel"
	if bus.EmitPlayerChat(p, &message) {
		t.Fatal("Java cancellation lost through IPC")
	}
	bus.Detach("test.chat")
	if err := loaded.Unload(t.Context()); err != nil {
		t.Fatal(err)
	}
}
