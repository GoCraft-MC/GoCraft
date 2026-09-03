package dispatch

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoCraft-MC/gocraft-abi/command"
)

func snapshotNames(root command.Root) []string {
	names := make([]string, 0, len(root.Children))
	for _, node := range root.Children {
		names = append(names, node.(command.Literal).Name)
	}
	return names
}

func TestSnapshotNamespacesPluginCollisions(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Source{Kind: SourceCore}, commandRoot("list", 1), commandHandlers(1)); err != nil {
		t.Fatal(err)
	}
	pluginA := command.Root{Children: []command.Node{
		command.Literal{Name: "shop", Exec: 1},
		command.Literal{Name: "warp", Exec: 2},
	}}
	handlersA := map[command.ExecID]Handler{
		1: func(context.Context, *Context) error { return nil },
		2: func(context.Context, *Context) error { return nil },
	}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "a"}, pluginA, handlersA); err != nil {
		t.Fatal(err)
	}
	pluginZ := command.Root{Children: []command.Node{command.Literal{Name: "shop", Permission: "shop.z", Exec: 1}}}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "z"}, pluginZ, commandHandlers(1)); err != nil {
		t.Fatal(err)
	}

	denied := registry.Snapshot(commandSender{})
	if denied.Version != 3 {
		t.Fatalf("snapshot version = %d, want 3", denied.Version)
	}
	if got, want := snapshotNames(denied.Root), []string{"list", "a:shop", "warp"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("denied snapshot = %v, want %v", got, want)
	}
	allowed := registry.Snapshot(commandSender{permitted: true})
	if got, want := snapshotNames(allowed.Root), []string{"list", "a:shop", "warp", "z:shop"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("allowed snapshot = %v, want %v", got, want)
	}
	allowed.Root.Children[0] = command.Literal{Name: "changed", Exec: 99}
	if got := snapshotNames(registry.Snapshot(commandSender{}).Root)[0]; got != "list" {
		t.Fatalf("snapshot mutation changed the registry: %q", got)
	}
}

func TestSnapshotPrunesEmptyDeniedBranches(t *testing.T) {
	registry := NewRegistry()
	root := command.Root{Children: []command.Node{command.Literal{Name: "admin", Children: []command.Node{
		command.Literal{Name: "reload", Permission: "admin.reload", Exec: 1},
	}}}}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "admin"}, root, commandHandlers(1)); err != nil {
		t.Fatal(err)
	}
	if got := registry.Snapshot(commandSender{}).Root.Children; len(got) != 0 {
		t.Fatalf("denied branch remains visible: %#v", got)
	}
}
