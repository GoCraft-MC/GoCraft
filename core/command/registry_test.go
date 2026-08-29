package command

import (
	"context"
	"strings"
	"testing"
)

func commandRoot(name string, executor ExecID) Root {
	return Root{Children: []Node{Literal{Name: name, Exec: executor}}}
}

func commandHandlers(executor ExecID) map[ExecID]Handler {
	return map[ExecID]Handler{executor: func(context.Context, *Context) error { return nil }}
}

func TestRegistryRefusesCoreCommandOverride(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Source{Kind: SourceCore}, commandRoot("tp", 1), commandHandlers(1)); err != nil {
		t.Fatal(err)
	}
	err := registry.Register(Source{Kind: SourcePlugin, PluginID: "shop"}, commandRoot("tp", 2), commandHandlers(2))
	if err == nil || !strings.Contains(err.Error(), "conflicts with a core command") {
		t.Fatalf("Register() error = %v", err)
	}
	if registry.Version() != 1 {
		t.Fatalf("version changed after rejected registration: %d", registry.Version())
	}
}

func TestRegistryAllowsPluginCollisionForNamespacing(t *testing.T) {
	registry := NewRegistry()
	for _, tc := range []struct {
		id   string
		exec ExecID
	}{{"shop-one", 1}, {"shop-two", 2}} {
		err := registry.Register(Source{Kind: SourcePlugin, PluginID: tc.id}, commandRoot("shop", tc.exec), commandHandlers(tc.exec))
		if err != nil {
			t.Fatalf("Register(%s): %v", tc.id, err)
		}
	}
	if registry.Version() != 2 {
		t.Fatalf("version = %d, want 2", registry.Version())
	}
	registry.RevokeAll("shop-one")
	if registry.Version() != 3 {
		t.Fatalf("version after revoke = %d, want 3", registry.Version())
	}
	registry.RevokeAll("missing")
	if registry.Version() != 3 {
		t.Fatal("missing revoke changed the version")
	}
}

func TestRegistryRequiresOneHandlerPerExecutor(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(Source{Kind: SourceCore}, commandRoot("list", 4), nil)
	if err == nil || !strings.Contains(err.Error(), "has no handler") {
		t.Fatalf("Register() error = %v", err)
	}
	extra := commandHandlers(4)
	extra[5] = func(context.Context, *Context) error { return nil }
	err = registry.Register(Source{Kind: SourceCore}, commandRoot("list", 4), extra)
	if err == nil || !strings.Contains(err.Error(), "handler count") {
		t.Fatalf("Register() error = %v", err)
	}
}

func TestRegistryRemapsLocalExecutorIDs(t *testing.T) {
	registry := NewRegistry()
	for _, id := range []string{"one", "two"} {
		err := registry.Register(Source{Kind: SourcePlugin, PluginID: id}, commandRoot(id, 1), commandHandlers(1))
		if err != nil {
			t.Fatalf("Register(%s): %v", id, err)
		}
	}
	one := registry.entries["one"].root.Children[0].(Literal).Exec
	two := registry.entries["two"].root.Children[0].(Literal).Exec
	if one == 0 || two == 0 || one == two {
		t.Fatalf("global executors = %d, %d", one, two)
	}
	if registry.handlers[one].source.PluginID != "one" || registry.handlers[two].source.PluginID != "two" {
		t.Fatal("global handlers were assigned to the wrong sources")
	}
	registry.RevokeAll("one")
	if _, ok := registry.handlers[one]; ok {
		t.Fatal("revoked plugin handler remains registered")
	}
	if _, ok := registry.handlers[two]; !ok {
		t.Fatal("another plugin handler was revoked")
	}
}

func TestRegistryDoesNotRevokeCoreAsPlugin(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Source{Kind: SourceCore}, commandRoot("list", 1), commandHandlers(1)); err != nil {
		t.Fatal(err)
	}
	registry.RevokeAll("core")
	if _, ok := registry.entries["core"]; !ok {
		t.Fatal("core command source was revoked")
	}
	if registry.Version() != 1 {
		t.Fatalf("version = %d, want 1", registry.Version())
	}
}
