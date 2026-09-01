package handler

import (
	"context"
	"sort"
	"testing"

	"GoCraft/core/command"
	"GoCraft/core/player"
)

type permissiveSender struct{}

func (permissiveSender) Name() string                   { return "Console" }
func (permissiveSender) UUID() [16]byte                 { return [16]byte{} }
func (permissiveSender) Has(string) bool                { return true }
func (permissiveSender) SendMessage(string) error       { return nil }
func (permissiveSender) Player() (*player.Player, bool) { return nil, false }

func builtinRegistry(t *testing.T) (*Dispatcher, *command.Registry) {
	t.Helper()
	dispatcher := NewDispatcher()
	RegisterBuiltins(dispatcher)
	registry := command.NewRegistry()
	dispatcher.SetCommandRegistry(registry)
	return dispatcher, registry
}

func treeNames(root command.Root) []string {
	names := make([]string, 0, len(root.Children))
	for _, node := range root.Children {
		names = append(names, node.(command.Literal).Name)
	}
	sort.Strings(names)
	return names
}

// The tree has to hold every command the dispatcher answers for. A command that
// is dispatchable and unadvertised can only be found by being told about it.
func TestBuiltinTreeHoldsEveryRegisteredCommand(t *testing.T) {
	dispatcher, registry := builtinRegistry(t)

	dispatcher.mu.RLock()
	registered := make([]string, 0, len(dispatcher.cmds))
	for name := range dispatcher.cmds {
		registered = append(registered, name)
	}
	dispatcher.mu.RUnlock()
	sort.Strings(registered)

	published := treeNames(registry.Snapshot(permissiveSender{}).Root)
	if len(published) != len(registered) {
		t.Fatalf("tree holds %d commands, dispatcher answers for %d:\n tree %v\n cmds %v",
			len(published), len(registered), published, registered)
	}
	for index := range registered {
		if published[index] != registered[index] {
			t.Fatalf("tree = %v, dispatcher = %v", published, registered)
		}
	}
}

// The structures the hand-built graph carried have to survive the move, or tab
// completion quietly gets worse for every command that had any.
func TestBuiltinTreeKeepsItsArgumentStructures(t *testing.T) {
	_, registry := builtinRegistry(t)
	root := registry.Snapshot(permissiveSender{}).Root

	find := func(nodes []command.Node, name string) command.Node {
		t.Helper()
		for _, node := range nodes {
			switch typed := node.(type) {
			case command.Literal:
				if typed.Name == name {
					return typed
				}
			case command.Argument:
				if typed.Name == name {
					return typed
				}
			}
		}
		t.Fatalf("no node named %q", name)
		return nil
	}

	gamemode := find(root.Children, "gamemode").(command.Literal)
	if gamemode.Exec != 0 {
		t.Fatal("/gamemode became runnable with no mode")
	}
	find(gamemode.Children, "creative")

	// /kill runs alone and on a target, which is what an executable node with
	// children is for.
	kill := find(root.Children, "kill").(command.Literal)
	if kill.Exec == 0 {
		t.Fatal("/kill stopped being runnable on its own")
	}
	if find(kill.Children, "player").(command.Argument).Type != command.ArgPlayer {
		t.Fatal("/kill lost its player argument")
	}

	give := find(root.Children, "give").(command.Literal)
	item := find(find(give.Children, "player").(command.Argument).Children, "item").(command.Argument)
	count := find(item.Children, "count").(command.Argument)
	if count.IntegerMin == nil || *count.IntegerMin != 1 || count.IntegerMax == nil || *count.IntegerMax != 64 {
		t.Fatalf("/give count bounds = %+v", count)
	}

	// A command nobody described advertises the rest of the line, which is the
	// shape those already had.
	spawnBoat := find(root.Children, "spawnboat").(command.Literal)
	if find(spawnBoat.Children, "arguments").(command.Argument).Type != command.ArgGreedy {
		t.Fatal("/spawnboat lost its free-form argument")
	}
}

// Every executable node of one command reaches that command's handler, because
// the dispatcher routes on the name and the node only says what may be typed.
func TestBuiltinTreeGivesEachCommandOneExecutor(t *testing.T) {
	dispatcher, _ := builtinRegistry(t)
	root, handlers := dispatcher.commandTree()
	for _, node := range root.Children {
		literal := node.(command.Literal)
		executors := command.Executors(command.Root{Children: []command.Node{literal}})
		if len(executors) != 1 {
			t.Fatalf("/%s has %d executors", literal.Name, len(executors))
		}
		if handlers[executors[0]] == nil {
			t.Fatalf("/%s has no handler", literal.Name)
		}
	}
}

// A plugin cannot take a built-in name once the tree is published. Before it
// was, /tp was there for the taking.
func TestPublishedTreeProtectsBuiltinNames(t *testing.T) {
	_, registry := builtinRegistry(t)
	root := command.Root{Children: []command.Node{command.Literal{Name: "tp", Exec: 1}}}
	handlers := map[command.ExecID]command.Handler{
		1: func(ctx context.Context, call *command.Context) error { return nil },
	}
	if err := registry.Register(command.Source{Kind: command.SourcePlugin, PluginID: "x"}, root, handlers); err == nil {
		t.Fatal("a plugin took /tp")
	}
}

// A command registered later reaches clients: the tree is republished and the
// version moves, which is what tells both adapters to resend.
func TestRegisteringACommandRepublishesTheTree(t *testing.T) {
	dispatcher, registry := builtinRegistry(t)
	before := registry.Version()
	dispatcher.Register("latecomer", func(CommandContext) error { return nil })
	if registry.Version() <= before {
		t.Fatal("a late registration did not move the version")
	}
	found := false
	for _, name := range treeNames(registry.Snapshot(permissiveSender{}).Root) {
		found = found || name == "latecomer"
	}
	if !found {
		t.Fatal("a late registration never reached the tree")
	}
}

// serverCommandTree is the graph a connected player receives: the built-ins the
// handler package registers, plus the ones the server adds once it exists.
//
// The packet tests used to read a graph written by hand in this package, which
// listed commands nothing here registers. Rendering from the tree means the
// test has to register them, which is the point: the graph now says what the
// server actually answers for.
func serverCommandTree(t *testing.T) command.Root {
	t.Helper()
	dispatcher := NewDispatcher()
	RegisterBuiltins(dispatcher)
	for _, name := range []string{"timings", "tps", "mspt", "spawn", "setspawn", "gocraft"} {
		dispatcher.Register(name, func(CommandContext) error { return nil })
	}
	registry := command.NewRegistry()
	dispatcher.SetCommandRegistry(registry)
	return registry.Snapshot(permissiveSender{}).Root
}
