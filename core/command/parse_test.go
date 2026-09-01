package command

import (
	"errors"
	"strings"
	"testing"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func decimal(value float64) *float64 { return &value }

func integer(value int64) *int64 { return &value }

// shopTree is the worked example from §07: one command, a literal branch, a
// bounded decimal and a permission-guarded sibling.
func shopTree() Root {
	return Root{Children: []Node{Literal{Name: "shop", Children: []Node{
		Literal{Name: "sell", Children: []Node{
			Argument{
				Name: "price", Type: ArgDecimal,
				DecimalMin: decimal(0.01), DecimalMax: decimal(1000),
				Exec: 1,
			},
		}},
		Literal{Name: "admin", Permission: "shop.admin", Children: []Node{
			Literal{Name: "reload", Exec: 2},
		}},
	}}}}
}

func shopSnapshot(t *testing.T, sender Sender) Snapshot {
	t.Helper()
	registry := NewRegistry()
	handlers := map[ExecID]Handler{1: commandHandlers(1)[1], 2: commandHandlers(2)[2]}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "shop"}, shopTree(), handlers); err != nil {
		t.Fatal(err)
	}
	return registry.Snapshot(sender)
}

func TestResolveReachesTheExecutorWithItsArguments(t *testing.T) {
	snapshot := shopSnapshot(t, commandSender{})
	executor, values, err := snapshot.Resolve("/shop sell 12.5", Resolvers{})
	if err != nil {
		t.Fatal(err)
	}
	if executor == 0 {
		t.Fatal("resolve reached no executor")
	}
	price, ok := values.Decimal("price")
	if !ok || price != 12.5 {
		t.Fatalf("price = %v (%t), want 12.5", price, ok)
	}
}

func TestResolveAcceptsALineWithoutASlash(t *testing.T) {
	snapshot := shopSnapshot(t, commandSender{})
	if _, _, err := snapshot.Resolve("shop sell 1", Resolvers{}); err != nil {
		t.Fatal(err)
	}
}

// A branch the sender cannot use is absent from their snapshot, so the line
// reads as an unknown command. Reporting a denial instead would let anyone map
// a plugin's admin commands by typing them.
func TestResolveHidesBranchesTheSenderCannotUse(t *testing.T) {
	denied := shopSnapshot(t, commandSender{})
	if _, _, err := denied.Resolve("/shop admin reload", Resolvers{}); err == nil {
		t.Fatal("denied sender reached the admin branch")
	} else if !strings.Contains(err.Error(), "expected sell") {
		t.Fatalf("denied sender was told %q", err)
	}
	allowed := shopSnapshot(t, commandSender{permitted: true})
	if _, _, err := allowed.Resolve("/shop admin reload", Resolvers{}); err != nil {
		t.Fatal(err)
	}
}

func TestResolveReportsAnUnknownCommandDistinctly(t *testing.T) {
	snapshot := shopSnapshot(t, commandSender{})
	for _, line := range []string{"/warp home", "", "   "} {
		if _, _, err := snapshot.Resolve(line, Resolvers{}); !errors.Is(err, ErrNoSuchCommand) {
			t.Fatalf("resolve(%q) = %v, want ErrNoSuchCommand", line, err)
		}
	}
}

// The failure a matched argument produced beats "unexpected token": it is the
// only one whoever typed the line can act on.
func TestResolvePrefersTheInformativeFailure(t *testing.T) {
	snapshot := shopSnapshot(t, commandSender{})
	_, _, err := snapshot.Resolve("/shop sell 9000", Resolvers{})
	if err == nil || !strings.Contains(err.Error(), "between 0.01 and 1000") {
		t.Fatalf("out-of-range price reported as %v", err)
	}
	_, _, err = snapshot.Resolve("/shop sell cheap", Resolvers{})
	if err == nil || !strings.Contains(err.Error(), "must be a number") {
		t.Fatalf("unparsable price reported as %v", err)
	}
}

func TestResolveReportsIncompleteAndOverlongLines(t *testing.T) {
	snapshot := shopSnapshot(t, commandSender{})
	_, _, err := snapshot.Resolve("/shop sell", Resolvers{})
	if err == nil || !strings.Contains(err.Error(), "incomplete command, expected <price>") {
		t.Fatalf("incomplete line reported as %v", err)
	}
	_, _, err = snapshot.Resolve("/shop sell 1 2", Resolvers{})
	if err == nil || !strings.Contains(err.Error(), "too many arguments") {
		t.Fatalf("overlong line reported as %v", err)
	}
}

// A literal and an argument sharing a level is the case that needs backtracking:
// "home" is both a valid warp name and a literal, and only the tokens after it
// say which branch was meant.
func TestResolveBacktracksBetweenSiblings(t *testing.T) {
	registry := NewRegistry()
	root := Root{Children: []Node{Literal{Name: "warp", Children: []Node{
		Literal{Name: "home", Children: []Node{
			Literal{Name: "set", Exec: 1},
		}},
		Argument{Name: "target", Type: ArgString, Exec: 2},
	}}}}
	handlers := map[ExecID]Handler{1: commandHandlers(1)[1], 2: commandHandlers(2)[2]}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "warp"}, root, handlers); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot(commandSender{})

	set, _, err := snapshot.Resolve("/warp home set", Resolvers{})
	if err != nil {
		t.Fatal(err)
	}
	target, values, err := snapshot.Resolve("/warp home", Resolvers{})
	if err != nil {
		t.Fatal(err)
	}
	if set == target {
		t.Fatal("both lines reached the same executor")
	}
	if name, ok := values.String("target"); !ok || name != "home" {
		t.Fatalf("target = %q (%t), want home", name, ok)
	}
}

// A branch abandoned after it wrote an argument must leave nothing behind.
func TestResolveKeepsNoArgumentFromAFailedBranch(t *testing.T) {
	registry := NewRegistry()
	root := Root{Children: []Node{Literal{Name: "pay", Children: []Node{
		Argument{Name: "amount", Type: ArgInteger, Children: []Node{
			Literal{Name: "to", Exec: 1},
		}},
		Argument{Name: "everyone", Type: ArgString, Exec: 2},
	}}}}
	handlers := map[ExecID]Handler{1: commandHandlers(1)[1], 2: commandHandlers(2)[2]}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "pay"}, root, handlers); err != nil {
		t.Fatal(err)
	}
	_, values, err := registry.Snapshot(commandSender{}).Resolve("/pay 5", Resolvers{})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := values["amount"]; present {
		t.Fatalf("abandoned branch left amount behind: %v", values)
	}
	if word, ok := values.String("everyone"); !ok || word != "5" {
		t.Fatalf("everyone = %q (%t), want 5", word, ok)
	}
}

func TestParseArgumentReadsEveryDeclaredType(t *testing.T) {
	target := &player.Player{Username: "oreo"}
	resolvers := Resolvers{
		Player: func(name string) (*player.Player, bool) { return target, name == "oreo" },
		Block: func(id string) (coreworld.Block, bool) {
			return coreworld.Block{Namespace: "minecraft", Name: "stone"}, id == "minecraft:stone"
		},
		Item: func(id string) (player.ItemStack, bool) {
			return player.ItemStack{ItemID: id, Count: 1}, id == "minecraft:diamond"
		},
	}
	cases := []struct {
		name     string
		argument Argument
		tokens   []string
		consumed int
		check    func(Value) bool
	}{
		{"integer", Argument{Name: "n", Type: ArgInteger, IntegerMin: integer(1), IntegerMax: integer(9)},
			[]string{"4"}, 1, func(v Value) bool { return v.Integer == 4 }},
		{"string", Argument{Name: "s", Type: ArgString}, []string{"one", "two"}, 1,
			func(v Value) bool { return v.String == "one" }},
		{"greedy", Argument{Name: "s", Type: ArgGreedy}, []string{"hello", "there"}, 2,
			func(v Value) bool { return v.String == "hello there" }},
		{"enum", Argument{Name: "e", Type: ArgEnum, Enum: []string{"red", "blue"}}, []string{"blue"}, 1,
			func(v Value) bool { return v.String == "blue" }},
		{"player", Argument{Name: "p", Type: ArgPlayer}, []string{"oreo"}, 1,
			func(v Value) bool { return v.Player == target }},
		{"position", Argument{Name: "at", Type: ArgBlockPos}, []string{"10", "-64", "3"}, 3,
			func(v Value) bool { return v.Position == spatial.BlockPos{X: 10, Y: -64, Z: 3} }},
		{"block", Argument{Name: "b", Type: ArgBlockState}, []string{"minecraft:stone"}, 1,
			func(v Value) bool { return v.Block.ResourceLocation() == "minecraft:stone" }},
		{"item", Argument{Name: "i", Type: ArgItem}, []string{"minecraft:diamond"}, 1,
			func(v Value) bool { return v.Item.ItemID == "minecraft:diamond" }},
		{"duration seconds", Argument{Name: "d", Type: ArgDuration}, []string{"30s"}, 1,
			func(v Value) bool { return v.Duration == 30*time.Second }},
		{"duration ticks", Argument{Name: "d", Type: ArgDuration}, []string{"40"}, 1,
			func(v Value) bool { return v.Duration == 2*time.Second }},
		{"duration tick suffix", Argument{Name: "d", Type: ArgDuration}, []string{"20t"}, 1,
			func(v Value) bool { return v.Duration == time.Second }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			consumed, value, err := parseArgument(testCase.argument, testCase.tokens, resolvers)
			if err != nil {
				t.Fatal(err)
			}
			if consumed != testCase.consumed {
				t.Fatalf("consumed %d tokens, want %d", consumed, testCase.consumed)
			}
			if !testCase.check(value) {
				t.Fatalf("value = %+v", value)
			}
		})
	}
}

// An argument type the host offers no lookup for is refused, never resolved to
// something invented: a handler reading a made-up player is worse than a
// command that says it cannot run.
func TestParseArgumentRefusesWhatItCannotResolve(t *testing.T) {
	for _, kind := range []ArgType{ArgPlayer, ArgBlockState, ArgItem} {
		argument := Argument{Name: "target", Type: kind}
		if _, _, err := parseArgument(argument, []string{"anything"}, Resolvers{}); err == nil {
			t.Fatalf("argument type %d resolved with no resolver installed", kind)
		}
	}
}

func TestParseArgumentRefusesANegativeDuration(t *testing.T) {
	argument := Argument{Name: "d", Type: ArgDuration}
	for _, raw := range []string{"-5s", "-20"} {
		if _, _, err := parseArgument(argument, []string{raw}, Resolvers{}); err == nil {
			t.Fatalf("duration %q accepted", raw)
		}
	}
}

// An executable node with children is how an optional argument is spelled.
// Both /kill and /kill <player> have to reach an executor, and they are not the
// same one.
func TestResolveHandlesAnOptionalArgument(t *testing.T) {
	registry := NewRegistry()
	root := Root{Children: []Node{Literal{Name: "kill", Exec: 1, Children: []Node{
		Argument{Name: "target", Type: ArgPlayer, Exec: 2},
	}}}}
	handlers := map[ExecID]Handler{1: commandHandlers(1)[1], 2: commandHandlers(2)[2]}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "kill"}, root, handlers); err != nil {
		t.Fatal(err)
	}
	target := &player.Player{Username: "oreo"}
	resolvers := Resolvers{Player: func(string) (*player.Player, bool) { return target, true }}
	snapshot := registry.Snapshot(commandSender{})

	bare, values, err := snapshot.Resolve("/kill", resolvers)
	if err != nil || len(values) != 0 {
		t.Fatalf("bare form = %v, %v", values, err)
	}
	aimed, values, err := snapshot.Resolve("/kill oreo", resolvers)
	if err != nil {
		t.Fatal(err)
	}
	if bare == aimed {
		t.Fatal("both forms reached the same executor")
	}
	if values["target"].Player != target {
		t.Fatalf("target = %+v", values["target"])
	}
}
