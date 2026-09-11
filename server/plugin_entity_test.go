package server

import (
	"context"
	"testing"
	"time"

	"GoCraft/core/entity"
	coreplugin "GoCraft/core/plugin"
	coreworld "GoCraft/core/world"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestEntityDamageFiltersBeforeQueueCoalescingAcrossDimensions(t *testing.T) {
	for _, dimension := range []int32{dimensionOverworld, dimensionNether, dimensionEnd} {
		w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
		mob := &entity.Entity{EntityID: 1, Type: entity.TypeCow, Health: 10, MaxHealth: 10}
		w.Entities.Add(mob)
		cancel, calls := true, 0
		bus := coreplugin.NewBus(context.Background(), time.Second)
		if err := bus.Attach(serverEventPlugin{event: coreplugin.EventEntityDamage, handle: func(e *abi.Event) abi.Verdict {
			calls++
			if e.Fields[4].Int64 != int64(dimension) {
				t.Fatal("wrong dimension")
			}
			return abi.Verdict{Cancelled: cancel, Mutations: []abi.Mutation{{Path: []uint32{2}, Value: abi.Double(2)}}}
		}}); err != nil {
			t.Fatal(err)
		}
		s := &Server{plugins: bus}
		s.installWorldEvents(w, dimension)
		if w.QueueEntityDamage(1, 5) || len(w.DrainEntityDamage()) != 0 {
			t.Fatal("cancelled hit queued")
		}
		cancel = false
		w.QueueEntityDamage(1, 5)
		w.QueueEntityDamage(1, 5)
		if hit := w.DrainEntityDamage()[1]; hit.Amount != 4 || calls != 3 {
			t.Fatalf("amount=%v calls=%d", hit.Amount, calls)
		}
		simulation := s.dimensionSimulation(dimension, w)
		cancel = true
		if simulation.damageEnvironmentalEntity(mob, 3, "fire") || mob.Health != 10 {
			t.Fatal("dimension simulation lost event bus")
		}
		w.Close()
	}
}
