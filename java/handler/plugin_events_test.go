package handler

import (
	"context"
	"testing"
	"time"

	coreplugin "GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

type eventTestPlugin struct {
	event  string
	handle func(*abi.Event) abi.Verdict
}

func (p eventTestPlugin) Manifest() gcpkg.Manifest {
	return gcpkg.Manifest{ID: "event-test", Subscriptions: []gcpkg.Subscription{{Event: p.event}}}
}
func (p eventTestPlugin) Dispatch(_ context.Context, e *abi.Event) (abi.Verdict, error) {
	return p.handle(e), nil
}
func (eventTestPlugin) Unload(context.Context) error { return nil }

func testEventBus(t *testing.T, event string, handle func(*abi.Event) abi.Verdict) *coreplugin.Bus {
	t.Helper()
	bus := coreplugin.NewBus(context.Background(), time.Second)
	if err := bus.Attach(eventTestPlugin{event: event, handle: handle}); err != nil {
		t.Fatal(err)
	}
	return bus
}
