package command

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExecuteRunsTheHandlerItResolved(t *testing.T) {
	registry := NewRegistry()
	var received Values
	root := Root{Children: []Node{Literal{Name: "shop", Children: []Node{
		Argument{Name: "price", Type: ArgDecimal, Exec: 1},
	}}}}
	handlers := map[ExecID]Handler{1: func(_ context.Context, call *Context) error {
		received = call.Args
		return nil
	}}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "shop"}, root, handlers); err != nil {
		t.Fatal(err)
	}

	handled, err := registry.Execute(context.Background(), commandSender{}, "/shop 4.5", Resolvers{})
	if !handled || err != nil {
		t.Fatalf("execute = (%t, %v), want (true, nil)", handled, err)
	}
	if price, ok := received.Decimal("price"); !ok || price != 4.5 {
		t.Fatalf("handler received %v", received)
	}
}

// A line this registry does not own is reported as unhandled rather than as an
// error, so the caller can offer it to whoever else dispatches commands.
func TestExecuteLeavesAnUnknownLineToTheCaller(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "shop"},
		commandRoot("shop", 1), commandHandlers(1)); err != nil {
		t.Fatal(err)
	}
	handled, err := registry.Execute(context.Background(), commandSender{}, "/gamemode creative", Resolvers{})
	if handled || err != nil {
		t.Fatalf("execute = (%t, %v), want (false, nil)", handled, err)
	}
}

func TestExecuteReportsAResolutionFailureAsItsOwn(t *testing.T) {
	registry := NewRegistry()
	root := Root{Children: []Node{Literal{Name: "shop", Children: []Node{
		Argument{Name: "price", Type: ArgDecimal, Exec: 1},
	}}}}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "shop"}, root, commandHandlers(1)); err != nil {
		t.Fatal(err)
	}
	handled, err := registry.Execute(context.Background(), commandSender{}, "/shop cheap", Resolvers{})
	if !handled {
		t.Fatal("a line naming a known command was reported as unhandled")
	}
	if err == nil || !strings.Contains(err.Error(), "must be a number") {
		t.Fatalf("execute error = %v", err)
	}
}

// The snapshot hides what the sender cannot use, so an executor they reached
// another way is still refused by Invoke. Both checks exist because neither
// alone survives a change to the other.
func TestExecuteStillChecksPermissionOnInvoke(t *testing.T) {
	registry := NewRegistry()
	root := Root{Children: []Node{Literal{
		Name: "admin", Permission: "server.admin", Children: []Node{Literal{Name: "reload", Exec: 1}},
	}}}
	if err := registry.Register(Source{Kind: SourcePlugin, PluginID: "admin"}, root, commandHandlers(1)); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Snapshot(commandSender{permitted: true})
	executor, _, err := snapshot.Resolve("/admin reload", Resolvers{})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Invoke(context.Background(), executor, commandSender{}, nil); !errors.Is(err, ErrPermission) {
		t.Fatalf("invoke by a denied sender = %v, want ErrPermission", err)
	}
}
