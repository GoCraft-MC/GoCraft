package goplugin

import (
	"context"
	"os"
	"testing"
	"time"

	"GoCraft/core/command"
	"GoCraft/core/player"
	"GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	wire "github.com/GoCraft-MC/gocraft-abi/abi/v1/wire"

	"google.golang.org/protobuf/proto"
)

// commandTreeEntry is where the test bundle keeps its tree, as a manifest would
// name it.
const commandTreeEntry = "commands.pb"

// helperCommandTree declares the one command the helper plugin registers. The
// executor id lives here and nowhere else: the plugin binds to the path.
func helperCommandTree(t *testing.T) []byte {
	t.Helper()
	encoded, err := proto.Marshal(&wire.CommandTree{Version: 1, Children: []*wire.CommandNode{{
		Kind: wire.CommandNodeKind_COMMAND_NODE_KIND_LITERAL, Name: "give",
		Children: []*wire.CommandNode{{
			Kind:         wire.CommandNodeKind_COMMAND_NODE_KIND_ARGUMENT,
			Name:         "amount",
			ArgumentType: wire.CommandArgumentType_COMMAND_ARGUMENT_TYPE_INTEGER,
			Executor:     7,
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

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
		Path:          writeTestBundleWith(t, "bin/example", []byte("placeholder"), helperCommandTree(t)),
		DataDirectory: t.TempDir(),
		Manifest: plugin.Manifest{
			ID: "example", Version: "1.0.0", APIVersion: 1,
			Runtime: RuntimeName, Entry: "bin/example",
			CommandTree:   commandTreeEntry,
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
	result, err := commands.InvokeCommand(t.Context(), 7, sender, command.Values{
		"amount": {Type: command.ArgInteger, Integer: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != "" {
		t.Fatalf("InvokeCommand() error = %q", result.Error)
	}
	// The reply comes back as an effect for the tick to deliver rather than
	// being written to the sender from this goroutine, which is what makes a
	// command handler reach the world by the same path an event handler does.
	if len(result.Effects) != 1 || result.Effects[0].Type != "chat.message" ||
		result.Effects[0].Fields[1].String != "amount=4" {
		t.Fatalf("command effects = %#v", result.Effects)
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
