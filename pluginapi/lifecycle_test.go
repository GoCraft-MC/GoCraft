package pluginapi

import (
	"errors"
	"testing"
)

type lifecyclePlugin struct {
	context               Context
	loadErr, enableErr    error
	loads, enables, stops int
}

func (p *lifecyclePlugin) OnLoad(context Context) error {
	p.loads++
	p.context = context
	if p.loadErr == nil {
		return context.Events().OnPlayerJoin(func(*PlayerJoinEvent) {})
	}
	return p.loadErr
}

func (p *lifecyclePlugin) OnEnable() error {
	p.enables++
	return p.enableErr
}

func (p *lifecyclePlugin) OnDisable() error {
	p.stops++
	return nil
}

func TestRuntimeStateLifecycle(t *testing.T) {
	implementation := &lifecyclePlugin{}
	state := newRuntimeState(Metadata{ID: "example", Version: "1", APIVersion: 1}, implementation)
	events, err := state.load("example", "plugin-data")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0] != EventPlayerJoin {
		t.Fatalf("load events = %v", events)
	}
	if implementation.context.DataDirectory() != "plugin-data" || !state.enabled {
		t.Fatal("load did not expose the context or enable the plugin")
	}
	if _, err := state.load("example", "plugin-data"); err == nil {
		t.Fatal("duplicate load was accepted")
	}
	if err := state.disable(); err != nil {
		t.Fatal(err)
	}
	if implementation.loads != 1 || implementation.enables != 1 || implementation.stops != 1 {
		t.Fatalf("lifecycle calls = %d, %d, %d", implementation.loads, implementation.enables, implementation.stops)
	}
}

func TestRuntimeStateFailuresCleanOwnership(t *testing.T) {
	loadFailure := &lifecyclePlugin{loadErr: errors.New("bad config")}
	state := newRuntimeState(Metadata{ID: "example"}, loadFailure)
	if _, err := state.load("example", "data"); err == nil || state.context != nil {
		t.Fatal("load failure retained plugin state")
	}
	if loadFailure.stops != 0 {
		t.Fatal("OnDisable ran before OnLoad completed")
	}

	enableFailure := &lifecyclePlugin{enableErr: errors.New("not ready")}
	state = newRuntimeState(Metadata{ID: "example"}, enableFailure)
	if _, err := state.load("example", "data"); err == nil || state.context != nil {
		t.Fatal("enable failure retained plugin state")
	}
	if enableFailure.stops != 1 {
		t.Fatal("enable failure did not disable initialized plugin")
	}
}
