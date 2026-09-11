package server

import (
	"context"
	"testing"
	"time"

	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	"GoCraft/java/handler"
	"GoCraft/java/session"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestDeathEventRunsOnceAndOnlyAfterTotemResolution(t *testing.T) {
	p := player.New([16]byte{1}, "Alex", player.ClientEditionJava)
	p.GameMode = player.GameModeCreative // /kill bypasses immunity; no drops in this mode.
	p.Inventory[player.OffhandSlot] = player.ItemStack{ItemID: "minecraft:totem_of_undying", Count: 1}
	bus := coreplugin.NewBus(context.Background(), time.Second)
	received := make(chan string, 4)
	if err := bus.Attach(serverEventPlugin{event: coreplugin.EventPlayerDeath, handle: func(e *abi.Event) abi.Verdict {
		received <- e.Fields[1].String
		return abi.Verdict{}
	}}); err != nil {
		t.Fatal(err)
	}
	s := &Server{plugins: bus}
	s.installPlayerEvents(p)
	target := &session.Session{Player: p}
	handler.KillPlayer(target, "test", nil)
	if p.Dead {
		t.Fatal("totem emitted death")
	}
	handler.KillPlayer(target, "test", nil)
	handler.KillPlayer(target, "test", nil)
	if !p.Dead {
		t.Fatal("death was prevented")
	}
	select {
	case cause := <-received:
		if cause != "test" {
			t.Fatal(cause)
		}
	case <-time.After(time.Second):
		t.Fatal("death event missing")
	}
	select {
	case <-received:
		t.Fatal("duplicate death event")
	case <-time.After(10 * time.Millisecond):
	}
}
