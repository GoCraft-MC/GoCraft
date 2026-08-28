package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	abi "GoCraft/abi/v1"
)

func TestEmitCancellableUsesPriorityOrder(t *testing.T) {
	bus := NewBus(context.Background(), time.Second)
	var calls []string
	for _, tc := range []struct {
		id       string
		priority Priority
		cancel   bool
	}{{"last", PriorityLow, false}, {"first", PriorityHigh, false}, {"stop", PriorityNormal, true}} {
		tc := tc
		instance := &fakeInstance{
			manifest: Manifest{ID: tc.id, Subscriptions: []Subscription{{Event: "block.break", Priority: tc.priority}}},
			dispatch: func(context.Context, *abi.Event) (abi.Verdict, error) {
				calls = append(calls, tc.id)
				return abi.Verdict{Cancelled: tc.cancel}, nil
			},
		}
		if err := bus.Attach(instance); err != nil {
			t.Fatal(err)
		}
	}
	if bus.EmitCancellable(&abi.Event{Type: "block.break"}) {
		t.Fatal("cancelled event was allowed")
	}
	if want := []string{"first", "stop"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestEmitCancellableAppliesFailurePolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy abi.FailurePolicy
		allow  bool
	}{{"allow", abi.FailureAllow, true}, {"deny", abi.FailureDeny, false}} {
		t.Run(tc.name, func(t *testing.T) {
			bus := NewBus(context.Background(), time.Second)
			instance := &fakeInstance{
				manifest: Manifest{ID: "broken", Subscriptions: []Subscription{{Event: "block.break"}}},
				dispatch: func(context.Context, *abi.Event) (abi.Verdict, error) {
					return abi.Verdict{}, errors.New("runtime stopped")
				},
			}
			if err := bus.Attach(instance); err != nil {
				t.Fatal(err)
			}
			if got := bus.EmitCancellable(&abi.Event{Type: "block.break", OnFailure: tc.policy}); got != tc.allow {
				t.Fatalf("EmitCancellable() = %v, want %v", got, tc.allow)
			}
		})
	}
}

func TestEventDeadlineStopsRemainingSubscribers(t *testing.T) {
	bus := NewBus(context.Background(), 5*time.Millisecond)
	lateCalled := false
	slow := &fakeInstance{
		manifest: Manifest{ID: "slow", Subscriptions: []Subscription{{Event: "block.break", Priority: PriorityHigh}}},
		dispatch: func(ctx context.Context, _ *abi.Event) (abi.Verdict, error) {
			<-ctx.Done()
			return abi.Verdict{}, ctx.Err()
		},
	}
	late := &fakeInstance{
		manifest: Manifest{ID: "late", Subscriptions: []Subscription{{Event: "block.break", Priority: PriorityLow}}},
		dispatch: func(context.Context, *abi.Event) (abi.Verdict, error) {
			lateCalled = true
			return abi.Verdict{}, nil
		},
	}
	for _, instance := range []*fakeInstance{slow, late} {
		if err := bus.Attach(instance); err != nil {
			t.Fatal(err)
		}
	}
	if !bus.EmitCancellable(&abi.Event{Type: "block.break", OnFailure: abi.FailureAllow}) {
		t.Fatal("fail-open deadline denied the event")
	}
	if lateCalled {
		t.Fatal("subscriber ran after the shared budget expired")
	}
}
