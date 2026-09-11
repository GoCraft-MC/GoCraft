package plugin

import (
	"context"
	"math"
	"testing"
	"time"

	"GoCraft/core/player"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

func TestNativeMutationIsAppliedBeforeTheNextSubscriber(t *testing.T) {
	bus := NewBus(context.Background(), time.Second)
	calls := 0
	for _, id := range []string{"first", "second"} {
		instance := &fakeInstance{manifest: gcpkg.Manifest{ID: id, Subscriptions: []gcpkg.Subscription{{Event: EventPlayerChat}}},
			dispatch: func(_ context.Context, event *abi.Event) (abi.Verdict, error) {
				calls++
				if calls == 2 && event.Fields[1].String != "rewritten" {
					t.Fatal("next subscriber did not see the first mutation")
				}
				return abi.Verdict{Mutations: []abi.Mutation{{Path: []uint32{1}, Value: abi.String("rewritten")}}}, nil
			}}
		if err := bus.Attach(instance); err != nil {
			t.Fatal(err)
		}
	}
	text := "original"
	if !bus.EmitPlayerChat(player.New([16]byte{1}, "Alex", player.ClientEditionJava), &text) || text != "rewritten" || calls != 2 {
		t.Fatalf("message=%q calls=%d", text, calls)
	}
}

func TestNativeMutationRejectsReadonlyNestedMalformedAndNonfiniteWrites(t *testing.T) {
	bus := NewBus(context.Background(), time.Second)
	sub := &subscriber{id: "rogue"}
	event := &abi.Event{Type: EventPlayerDamage, Fields: BlankEvent(EventPlayerDamage)}
	event.Fields[1] = abi.Double(4)
	for _, mutation := range []abi.Mutation{
		{Path: []uint32{0}, Value: abi.String("impersonate")},
		{Path: []uint32{3, 0, 1}, Value: abi.Bool(true)},
		{Path: []uint32{1}, Value: abi.String("wrong kind")},
		{Path: []uint32{1}, Value: abi.Double(math.NaN())},
		{Path: []uint32{1}, Value: abi.Double(math.Inf(1))},
		{Path: []uint32{999}, Value: abi.Double(1)},
	} {
		bus.applyNativeMutations(sub, event, []abi.Mutation{mutation})
	}
	if event.Fields[1].Double != 4 || event.Fields[0].Kind != abi.ValueList {
		t.Fatal("invalid mutations changed the event")
	}
	for _, eventType := range NativeEvents() {
		if len(BlankEvent(eventType)) == 0 {
			t.Errorf("%s has no shaped warm-up payload", eventType)
		}
	}
}
