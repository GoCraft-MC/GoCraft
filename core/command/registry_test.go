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
