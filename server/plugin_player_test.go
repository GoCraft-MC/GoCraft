package server

import (
	"context"
	"testing"
	"time"

	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
	"GoCraft/java/session"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestPlayerDamageRoundTripPreservesArmourAndAbsorptionOnCancel(t *testing.T) {
	for _, edition := range []player.ClientEdition{player.ClientEditionJava, player.ClientEditionBedrock} {
		p := player.New([16]byte{1}, "Alex", edition)
		p.GameMode = player.GameModeSurvival
		p.Absorption = 4
		p.Inventory[5] = player.ItemStack{ItemID: "minecraft:iron_helmet", Count: 1}
		cancel, calls := true, 0
		bus := coreplugin.NewBus(context.Background(), time.Second)
		if err := bus.Attach(serverEventPlugin{event: coreplugin.EventPlayerDamage, handle: func(e *abi.Event) abi.Verdict {
			p.HealthSnapshot() // The callback must not run with healthMu held.
			calls++
			return abi.Verdict{Cancelled: cancel, Mutations: []abi.Mutation{{Path: []uint32{1}, Value: abi.Double(6)}}}
		}}); err != nil {
			t.Fatal(err)
		}
		s := &Server{plugins: bus}
		s.installPlayerEvents(p)
		target := &session.Session{Player: p}
		if handler.DamagePlayer(target, 10, "test", nil) || p.Health != 20 || p.Absorption != 4 || p.Inventory[5].Damage != 0 {
			t.Fatal("cancelled damage changed player")
		}
		cancel = false
		if !handler.DamagePlayer(target, 10, "test", nil) || p.Health != 18 || p.Absorption != 0 || calls != 2 {
			t.Fatalf("mutation lost: health=%v absorption=%v calls=%d", p.Health, p.Absorption, calls)
		}
	}
}

func TestQuitAndRespawnEventsIgnoreDuplicateRequests(t *testing.T) {
	for _, event := range []string{coreplugin.EventPlayerQuit, coreplugin.EventPlayerRespawn} {
		w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
		g := game.New()
		p := player.New([16]byte{1}, "Alex", player.ClientEditionBedrock)
		if err := g.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
		received := make(chan struct{}, 4)
		bus := coreplugin.NewBus(context.Background(), time.Second)
		if err := bus.Attach(serverEventPlugin{event: event, handle: func(e *abi.Event) abi.Verdict { received <- struct{}{}; return abi.Verdict{Cancelled: true} }}); err != nil {
			t.Fatal(err)
		}
		s := &Server{plugins: bus, game: g, world: w}
		if event == coreplugin.EventPlayerQuit {
			s.unregisterPlayer(p.UUID, "test")
			s.unregisterPlayer(p.UUID, "duplicate")
			if g.GetPlayer(p.UUID) != nil {
				t.Fatal("observational quit prevented disconnect")
			}
		} else {
			p.ApplyDamage(20, "test")
			s.applyBedrockRespawn(intent.RespawnIntent{PlayerUUID: p.UUID})
			s.applyBedrockRespawn(intent.RespawnIntent{PlayerUUID: p.UUID})
			if p.Dead {
				t.Fatal("observational respawn was cancelled")
			}
		}
		w.Close()
		select {
		case <-received:
		case <-time.After(time.Second):
			t.Fatalf("%s missing", event)
		}
		select {
		case <-received:
			t.Fatalf("duplicate %s", event)
		case <-time.After(10 * time.Millisecond):
		}
	}
}
