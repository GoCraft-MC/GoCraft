package goplugin

import (
	"context"
	"os"
	"testing"
	"time"

	abi "GoCraft/abi/v1"
	"GoCraft/core/command"
	"GoCraft/core/player"
	"GoCraft/core/plugin"
)

type testSender struct{ messages []string }

func (*testSender) Name() string                   { return "Console" }
func (*testSender) UUID() [16]byte                 { return [16]byte{} }
func (*testSender) Has(string) bool                { return true }
func (*testSender) Player() (*player.Player, bool) { return nil, false }
func (s *testSender) SendMessage(message string) error {
	s.messages = append(s.messages, message)
	return nil
}

func TestRuntimeLoadsDispatchesCommandsAndStops(t *testing.T) {
	socketDirectory, err := os.MkdirTemp("", "gc-go-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDirectory)
	runtime := New(Config{
		ExtractDirectory: t.TempDir(), SocketDirectory: socketDirectory,
		StartTimeout: 3 * time.Second, Spawn: helperSpawn,
	})
	if err := runtime.Start(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	bundle := plugin.Bundle{
		Path:          writeTestBundle(t, "bin/example", []byte("placeholder")),
		DataDirectory: t.TempDir(),
		Manifest: plugin.Manifest{
			ID: "example", Version: "1.0.0", APIVersion: 1,
			Runtime: RuntimeName, Entry: "bin/example",
			Subscriptions: []plugin.Subscription{{Event: "block.break"}},
		},
	}
	loaded, err := runtime.Load(t.Context(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Ready(t.Context()); err != nil {
		t.Fatal(err)
	}
	verdict, err := loaded.Dispatch(t.Context(), runtimeBlockBreakEvent())
	if err != nil || !verdict.Cancelled {
		t.Fatalf("Dispatch() = %#v, %v", verdict, err)
	}
	sender := &testSender{}
	commands := loaded.(plugin.CommandInstance)
	if err := commands.InvokeCommand(t.Context(), 7, sender, command.Values{
		"amount": {Type: command.ArgInteger, Integer: 4},
	}); err != nil {
		t.Fatal(err)
	}
	if len(sender.messages) != 1 || sender.messages[0] != "amount=4" {
		t.Fatalf("command messages = %v", sender.messages)
	}
	if err := loaded.Unload(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func runtimeBlockBreakEvent() *abi.Event {
	actor := abi.List(abi.Bytes(make([]byte, 16)), abi.String("Elias"), abi.String("java"))
	return &abi.Event{Type: "block.break", OnFailure: abi.FailureAllow, Fields: []abi.Value{
		actor, abi.List(abi.Int64(1), abi.Int64(64), abi.Int64(2)),
		abi.List(abi.String("minecraft:stone"), abi.List()), abi.String("minecraft:pickaxe"), abi.List(),
	}}
}
