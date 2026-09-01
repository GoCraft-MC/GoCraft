package dispatch

import (
	"context"
	"errors"
	"testing"

	"GoCraft/core/player"

	"github.com/GoCraft-MC/gocraft-abi/command"
)

type commandSender struct{ permitted bool }

func (s commandSender) Name() string                   { return "tester" }
func (s commandSender) UUID() [16]byte                 { return [16]byte{1} }
func (s commandSender) Has(string) bool                { return s.permitted }
func (s commandSender) SendMessage(string) error       { return nil }
func (s commandSender) Player() (*player.Player, bool) { return nil, false }

func TestInvokeEnforcesInheritedPermission(t *testing.T) {
	registry := NewRegistry()
	called := false
	root := command.Root{Children: []command.Node{command.Literal{
		Name: "admin", Permission: "server.admin", Children: []command.Node{
			command.Argument{Name: "target", Type: command.ArgPlayer, Exec: 7},
		},
	}}}
	handlers := map[command.ExecID]Handler{7: func(_ context.Context, call *Context) error {
		called = true
		if call.Args["target"].String != "alex" {
			t.Fatal("parsed arguments were not passed to the handler")
		}
		return nil
	}}
	if err := registry.Register(Source{Kind: SourceCore}, root, handlers); err != nil {
		t.Fatal(err)
	}
	executor := registry.entries["core"].root.Children[0].(command.Literal).Children[0].(command.Argument).Exec
	args := Values{"target": {Type: command.ArgPlayer, String: "alex"}}
	if err := registry.Invoke(context.Background(), executor, commandSender{}, args); !errors.Is(err, ErrPermission) {
		t.Fatalf("Invoke() error = %v, want permission denied", err)
	}
	if called {
		t.Fatal("denied command reached its handler")
	}
	if err := registry.Invoke(context.Background(), executor, commandSender{permitted: true}, args); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("allowed command did not reach its handler")
	}
}

func TestInvokeRejectsUnknownExecutor(t *testing.T) {
	err := NewRegistry().Invoke(context.Background(), 42, commandSender{}, nil)
	if !errors.Is(err, ErrUnknownExecutor) {
		t.Fatalf("Invoke() error = %v", err)
	}
}
