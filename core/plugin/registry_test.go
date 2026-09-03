package plugin

import (
	"context"
	"reflect"
	"testing"

	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"

	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

type fakeInstance struct {
	manifest gcpkg.Manifest
	dispatch func(context.Context, *abi.Event) (abi.Verdict, error)
}

func (f *fakeInstance) Manifest() gcpkg.Manifest     { return f.manifest }
func (f *fakeInstance) Unload(context.Context) error { return nil }
func (f *fakeInstance) Dispatch(ctx context.Context, event *abi.Event) (abi.Verdict, error) {
	if f.dispatch == nil {
		return abi.Verdict{}, nil
	}
	return f.dispatch(ctx, event)
}

func TestAttachOrdersSubscriptions(t *testing.T) {
	bus := NewBus(context.Background(), 0)
	plugins := []*fakeInstance{
		{manifest: gcpkg.Manifest{ID: "low", Subscriptions: []gcpkg.Subscription{{Event: "block.break", Priority: gcpkg.PriorityLow}}}},
		{manifest: gcpkg.Manifest{ID: "bravo", Subscriptions: []gcpkg.Subscription{{Event: "block.break", Priority: gcpkg.PriorityHigh}}}},
		{manifest: gcpkg.Manifest{ID: "monitor", Subscriptions: []gcpkg.Subscription{{Event: "block.break", Priority: gcpkg.PriorityMonitor}}}},
		{manifest: gcpkg.Manifest{ID: "alpha", Subscriptions: []gcpkg.Subscription{{Event: "block.break", Priority: gcpkg.PriorityHigh}}}},
	}
	for _, instance := range plugins {
		if err := bus.Attach(instance); err != nil {
			t.Fatalf("Attach(%s): %v", instance.manifest.ID, err)
		}
	}
	var got []string
	for _, sub := range bus.subs["block.break"] {
		got = append(got, sub.id)
	}
	want := []string{"alpha", "bravo", "low", "monitor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subscription order = %v, want %v", got, want)
	}
}

func TestAttachRejectsDuplicateWithoutPartialRegistration(t *testing.T) {
	bus := NewBus(context.Background(), 0)
	instance := &fakeInstance{manifest: gcpkg.Manifest{ID: "shop", Subscriptions: []gcpkg.Subscription{
		{Event: "block.break"}, {Event: "block.break"},
	}}}
	if err := bus.Attach(instance); err == nil {
		t.Fatal("duplicate subscription was accepted")
	}
	if len(bus.subs) != 0 {
		t.Fatalf("failed attach left subscriptions: %v", bus.subs)
	}
}

func TestDetachRevokesPluginSubscriptions(t *testing.T) {
	bus := NewBus(context.Background(), 0)
	instance := &fakeInstance{manifest: gcpkg.Manifest{ID: "shop", Subscriptions: []gcpkg.Subscription{
		{Event: "block.break"}, {Event: "player.join"},
	}}}
	if err := bus.Attach(instance); err != nil {
		t.Fatal(err)
	}
	bus.Detach("shop")
	if len(bus.subs) != 0 {
		t.Fatalf("detach left subscriptions: %v", bus.subs)
	}
}
