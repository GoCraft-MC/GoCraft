package plugin

import (
	"reflect"
	"testing"
	"time"

	"GoCraft/core/command"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

type fakeSender struct {
	name   string
	player *player.Player
	held   map[string]bool
}

func (s *fakeSender) Name() string             { return s.name }
func (s *fakeSender) UUID() [16]byte           { return [16]byte{} }
func (s *fakeSender) SendMessage(string) error { return nil }

func (s *fakeSender) Has(permission string) bool { return s.held[permission] }

func (s *fakeSender) Player() (*player.Player, bool) {
	return s.player, s.player != nil
}

// Every argument type has to reach the wire as the shape the plugin API
// documents, because a runtime reads a fixed layout and a wrong one is a zero
// rather than a failure.
func TestNewCommandInvocationConvertsEveryArgumentType(t *testing.T) {
	tests := []struct {
		name  string
		value command.Value
		want  abi.Value
	}{
		{"integer", command.Value{Type: command.ArgInteger, Integer: 42}, abi.Int64(42)},
		{"decimal", command.Value{Type: command.ArgDecimal, Decimal: 1.5}, abi.Double(1.5)},
		{"string", command.Value{Type: command.ArgString, String: "spawn"}, abi.String("spawn")},
		{"greedy", command.Value{Type: command.ArgGreedy, String: "a b c"}, abi.String("a b c")},
		{"enum", command.Value{Type: command.ArgEnum, String: "deny"}, abi.String("deny")},
		{"custom", command.Value{Type: command.ArgCustom, String: "x"}, abi.String("x")},
		{
			"blockpos",
			command.Value{Type: command.ArgBlockPos, Position: spatial.BlockPos{X: 1, Y: -2, Z: 3}},
			abi.List(abi.Int64(1), abi.Int64(-2), abi.Int64(3)),
		},
		{
			"duration",
			command.Value{Type: command.ArgDuration, Duration: 2 * time.Second},
			abi.Int64(2000),
		},
		{
			"item",
			command.Value{Type: command.ArgItem, Item: player.ItemStack{ItemID: "minecraft:stone", Count: 3, Damage: 1}},
			abi.List(abi.String("minecraft:stone"), abi.Int64(3), abi.Int64(1)),
		},
		{
			"blockstate",
			command.Value{Type: command.ArgBlockState, Block: coreworld.Block{Name: "minecraft:stone"}},
			blockValue(coreworld.Block{Name: "minecraft:stone"}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := NewCommandInvocation(1, nil, command.Values{"arg": test.value}, nil)
			if err != nil {
				t.Fatalf("NewCommandInvocation() = %v", err)
			}
			if len(invocation.Arguments) != 1 {
				t.Fatalf("NewCommandInvocation() produced %d arguments", len(invocation.Arguments))
			}
			if got := invocation.Arguments[0].Value; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("NewCommandInvocation() value = %+v, want %+v", got, test.want)
			}
		})
	}
}

// The wire carries the declared type beside the value so a runtime asking for
// the wrong one is refused rather than handed a zero.
func TestNewCommandInvocationCarriesTheArgumentType(t *testing.T) {
	invocation, err := NewCommandInvocation(3, nil, command.Values{
		"radius": {Type: command.ArgInteger, Integer: 8},
	}, nil)
	if err != nil {
		t.Fatalf("NewCommandInvocation() = %v", err)
	}
	if invocation.Executor != 3 {
		t.Fatalf("NewCommandInvocation() executor = %d, want 3", invocation.Executor)
	}
	if got := invocation.Arguments[0].Type; got != abi.CommandArgumentInteger {
		t.Fatalf("NewCommandInvocation() type = %d, want %d", got, abi.CommandArgumentInteger)
	}
}

// An argument with no type never reaches the socket. It would arrive as
// UNSPECIFIED and the runtime would have to guess which field to read.
func TestNewCommandInvocationRefusesAnUntypedArgument(t *testing.T) {
	_, err := NewCommandInvocation(1, nil, command.Values{"broken": {}}, nil)
	if err == nil {
		t.Fatal("NewCommandInvocation() accepted an argument with no type")
	}
}

// Arguments come out in name order. The map they arrive in has none, so without
// this the same command serialises differently from one invocation to the next.
func TestNewCommandInvocationOrdersArgumentsByName(t *testing.T) {
	invocation, err := NewCommandInvocation(1, nil, command.Values{
		"zulu":  {Type: command.ArgString, String: "z"},
		"alpha": {Type: command.ArgString, String: "a"},
		"mike":  {Type: command.ArgString, String: "m"},
	}, nil)
	if err != nil {
		t.Fatalf("NewCommandInvocation() = %v", err)
	}
	want := []string{"alpha", "mike", "zulu"}
	for index, name := range want {
		if got := invocation.Arguments[index].Name; got != name {
			t.Fatalf("NewCommandInvocation() argument %d = %q, want %q", index, got, name)
		}
	}
}

// The sender is resolved before the invocation leaves, because the ABI has no
// message for asking afterwards.
func TestNewCommandInvocationResolvesTheSenderPermissions(t *testing.T) {
	sender := &fakeSender{
		name:   "oreo",
		player: player.New([16]byte{9}, "oreo", player.ClientEditionJava),
		held:   map[string]bool{"worldguard.region.define": true},
	}
	invocation, err := NewCommandInvocation(1, sender, nil, []string{
		"worldguard.region.remove", "worldguard.region.define", "worldguard.region.remove",
	})
	if err != nil {
		t.Fatalf("NewCommandInvocation() = %v", err)
	}
	if invocation.Sender.Name != "oreo" {
		t.Fatalf("NewCommandInvocation() sender name = %q", invocation.Sender.Name)
	}
	// Deduplicated and sorted, so the payload is the same for the same manifest.
	want := []abi.Value{
		abi.List(abi.String("worldguard.region.define"), abi.Bool(true)),
		abi.List(abi.String("worldguard.region.remove"), abi.Bool(false)),
	}
	if !reflect.DeepEqual(invocation.Sender.Permissions, want) {
		t.Fatalf("NewCommandInvocation() permissions = %+v, want %+v", invocation.Sender.Permissions, want)
	}
	if !invocation.Sender.Allowed("worldguard.region.define") {
		t.Fatal("Allowed() = false for a node the sender holds")
	}
	// A node nobody declared was never resolved, so it reads false rather than
	// reaching for the server the ABI cannot reach.
	if invocation.Sender.Allowed("worldguard.region.undeclared") {
		t.Fatal("Allowed() = true for a node the manifest never declared")
	}
}

// The console has no player. A handler has to be able to tell, and the empty
// PlayerRef is how every other message says so.
func TestNewCommandInvocationHandlesAConsoleSender(t *testing.T) {
	invocation, err := NewCommandInvocation(1, nil, nil, []string{"gocraft.admin"})
	if err != nil {
		t.Fatalf("NewCommandInvocation() = %v", err)
	}
	if _, ok := PlayerUUIDFrom(invocation.Sender.Player); ok {
		t.Fatal("NewCommandInvocation() produced a player for a console sender")
	}
	if invocation.Sender.Allowed("gocraft.admin") {
		t.Fatal("Allowed() = true for a console sender that was never asked")
	}
}

// A reply to the console travels as a chat.message with an empty PlayerRef,
// which the tick drops on purpose — for an event that ref means a piston, and
// nobody to tell. A command always has somebody to tell.
func TestConsoleReplyIsAnsweredThroughTheSender(t *testing.T) {
	sender := &fakeSender{name: "Console"}
	reply := abi.HostCall{
		Type:   EffectMessage,
		Fields: []abi.Value{playerReference(nil), abi.String("done")},
	}

	text, ok := consoleReply(reply, sender)
	if !ok || text != "done" {
		t.Fatalf("consoleReply() = %q, %v; want the line to send", text, ok)
	}
}

// A reply that names a player is queued like every other effect, so it reaches
// them on the tick rather than from whatever goroutine ran the handler.
func TestAPlayerReplyIsLeftForTheQueue(t *testing.T) {
	sender := &fakeSender{name: "oreo", player: player.New([16]byte{4}, "oreo", player.ClientEditionJava)}
	reply := abi.HostCall{
		Type:   EffectMessage,
		Fields: []abi.Value{playerReference(sender.player), abi.String("done")},
	}

	if _, ok := consoleReply(reply, sender); ok {
		t.Fatal("consoleReply() diverted a reply that names a player")
	}
}

// Anything that is not a reply is queued, whoever typed the command.
func TestOtherEffectsAreNeverDiverted(t *testing.T) {
	effect := abi.HostCall{Type: "world.setblock", Fields: []abi.Value{abi.String("x")}}

	if _, ok := consoleReply(effect, &fakeSender{name: "Console"}); ok {
		t.Fatal("consoleReply() diverted an effect that is not a reply")
	}
}
