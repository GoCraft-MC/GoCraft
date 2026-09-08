package handler

import (
	"testing"

	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestCommandEventRewritesBeforePermissionsAndRunsOnce(t *testing.T) {
	d := NewDispatcher()
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	calls, executions, checks := 0, 0, 0
	cancel := false
	d.SetEventBus(testEventBus(t, coreplugin.EventPlayerCommand, func(e *abi.Event) abi.Verdict {
		calls++
		return abi.Verdict{Cancelled: cancel, Mutations: []abi.Mutation{{Path: []uint32{1}, Value: abi.String("/guarded rewritten")}}}
	}))
	d.Register("guarded", func(ctx CommandContext) error {
		executions++
		if len(ctx.Args) != 1 || ctx.Args[0] != "rewritten" {
			t.Fatal(ctx.Args)
		}
		return nil
	})
	d.SetPermissionChecker(func(_ *player.Player, node string, _ bool) bool {
		checks++
		return node != "gocraft.command.guarded" || p.Operator
	})
	d.Dispatch("original", CommandContext{Player: p})
	if calls != 1 || executions != 0 || checks == 0 {
		t.Fatalf("calls=%d runs=%d checks=%d", calls, executions, checks)
	}
	p.Operator = true
	d.Dispatch("original", CommandContext{Player: p})
	if calls != 2 || executions != 1 {
		t.Fatalf("calls=%d runs=%d", calls, executions)
	}
	cancel = true
	d.Dispatch("original", CommandContext{Player: p})
	if calls != 3 || executions != 1 {
		t.Fatalf("cancel ignored: calls=%d runs=%d", calls, executions)
	}
}

func TestChatEventMutationCancellationAndValidation(t *testing.T) {
	for _, tc := range []struct {
		text            string
		cancel, allowed bool
	}{
		{"/guarded", false, true}, {"rewritten", true, false}, {"", false, false},
		{"invalid\nline", false, false}, {string([]byte{255}), false, false},
	} {
		t.Run(tc.text, func(t *testing.T) {
			d := NewDispatcher()
			calls := 0
			d.SetEventBus(testEventBus(t, coreplugin.EventPlayerChat, func(e *abi.Event) abi.Verdict {
				calls++
				return abi.Verdict{Cancelled: tc.cancel, Mutations: []abi.Mutation{{Path: []uint32{1}, Value: abi.String(tc.text)}}}
			}))
			message := "original"
			p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
			if allowed := d.FilterPlayerChat(p, &message); allowed != tc.allowed || calls != 1 {
				t.Fatalf("allowed=%v calls=%d", allowed, calls)
			}
			if tc.allowed && message != tc.text {
				t.Fatal(message)
			}
		})
	}
}
