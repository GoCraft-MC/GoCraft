package plugin

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// readyRuntime records when the load phase ended, relative to the plugins that
// had to be up before it.
type readyRuntime struct {
	recordingRuntime
	err error
}

func (r *readyRuntime) Ready(context.Context) error {
	*r.order = append(*r.order, "ready")
	return r.err
}

// A runtime told it is ready while the host is still loading would let its
// plugins act on a world the rest of them have not seen yet, so this runs once
// and last.
func TestLoadAllEndsTheLoadPhaseAfterEveryPlugin(t *testing.T) {
	var order []string
	registry := NewRegistry(context.Background(), 0, nil, nil)
	if err := registry.RegisterRuntime(&readyRuntime{recordingRuntime: recordingRuntime{order: &order}}); err != nil {
		t.Fatal(err)
	}
	bundles := []Bundle{testBundle("shop"), testBundle("bank")}

	if err := registry.LoadAll(context.Background(), bundles); err != nil {
		t.Fatalf("LoadAll() = %v", err)
	}
	want := []string{"start", "load:bank", "load:shop", "ready"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

// A runtime that cannot go live has not loaded anything usable, so the boot is
// rolled back rather than continued with a runtime stuck in its load phase.
func TestLoadAllRollsBackWhenAReadyFails(t *testing.T) {
	var order []string
	registry := NewRegistry(context.Background(), 0, nil, nil)
	failure := errors.New("the registry never came up")
	if err := registry.RegisterRuntime(&readyRuntime{
		recordingRuntime: recordingRuntime{order: &order},
		err:              failure,
	}); err != nil {
		t.Fatal(err)
	}

	err := registry.LoadAll(context.Background(), []Bundle{testBundle("shop")})
	if !errors.Is(err, failure) {
		t.Fatalf("LoadAll() = %v, want the ready failure", err)
	}
	if _, loaded := registry.Instance("shop"); loaded {
		t.Fatal("LoadAll() left a plugin loaded after the load phase failed")
	}
	if !strings.Contains(strings.Join(order, ","), "unload:shop") {
		t.Fatalf("order = %v, want the loaded plugin unloaded", order)
	}
}

// An in-process runtime has no phase to end, which is why Ready is a separate
// interface rather than a method half the backends would leave empty.
func TestLoadAllSkipsARuntimeWithNoLoadPhase(t *testing.T) {
	var order []string
	registry := NewRegistry(context.Background(), 0, nil, nil)
	if err := registry.RegisterRuntime(&recordingRuntime{order: &order}); err != nil {
		t.Fatal(err)
	}
	if err := registry.LoadAll(context.Background(), []Bundle{testBundle("shop")}); err != nil {
		t.Fatalf("LoadAll() = %v", err)
	}
	if strings.Contains(strings.Join(order, ","), "ready") {
		t.Fatalf("order = %v, want no ready on a runtime that does not implement it", order)
	}
}
