package handler

import (
	"testing"

	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestCommandTeleportAppliesMutationBeforeAdapterAndCancelsOnce(t *testing.T) {
	d := NewDispatcher()
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	cancel, calls, moved := false, 0, 0
	d.SetEventBus(testEventBus(t, coreplugin.EventPlayerTeleport, func(e *abi.Event) abi.Verdict {
		calls++
		return abi.Verdict{Cancelled: cancel, Mutations: []abi.Mutation{{Path: []uint32{4}, Value: abi.Double(42)}}}
	}))
	teleport := d.eventTeleport(p, func(x, y, z float64) error {
		moved++
		if x != 42 {
			t.Fatal(x)
		}
		return nil
	})
	if err := teleport(1, 2, 3); err != nil {
		t.Fatal(err)
	}
	cancel = true
	if err := teleport(1, 2, 3); err == nil || calls != 2 || moved != 1 {
		t.Fatalf("calls=%d moves=%d err=%v", calls, moved, err)
	}
}
