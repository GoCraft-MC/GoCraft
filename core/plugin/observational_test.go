package plugin

import (
	"context"
	"testing"
	"time"

	abi "GoCraft/abi/v1"
)

type channelHost struct{ calls chan abi.HostCall }

func (h channelHost) Enqueue(call abi.HostCall) error {
	h.calls <- call
	return nil
}

func TestObservationalEventDoesNotBlockOrSharePayload(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	host := channelHost{calls: make(chan abi.HostCall, 1)}
	bus := newBus(context.Background(), time.Second, host)
	instance := &fakeInstance{
		manifest: Manifest{ID: "audit", Subscriptions: []Subscription{{Event: "player.join"}}},
		dispatch: func(_ context.Context, event *abi.Event) (abi.Verdict, error) {
			close(started)
			<-release
			event.Fields[0].Bytes[0] = 'j'
			return abi.Verdict{Effects: []abi.HostCall{{Type: "observed"}}}, nil
		},
	}
	if err := bus.Attach(instance); err != nil {
		t.Fatal(err)
	}
	event := &abi.Event{Type: "player.join", Fields: []abi.Value{abi.Bytes([]byte("hello"))}}
	bus.EmitObservational(event)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("observational subscriber did not start")
	}
	select {
	case <-host.calls:
		t.Fatal("EmitObservational waited for a blocked subscriber")
	default:
	}
	close(release)
	select {
	case call := <-host.calls:
		if call.Type != "observed" {
			t.Fatalf("effect type = %q", call.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("observational effect was not queued")
	}
	if got := string(event.Fields[0].Bytes); got != "hello" {
		t.Fatalf("subscriber mutated caller payload to %q", got)
	}
	health, _ := bus.Health("audit")
	if health.Calls != 1 || health.Failures != 0 {
		t.Fatalf("health = %+v", health)
	}
}
