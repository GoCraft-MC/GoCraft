package plugin

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"

	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"

	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

type fakeRuntime struct {
	name       string
	provisions atomic.Int32
}

func (r *fakeRuntime) Name() string { return r.name }
func (r *fakeRuntime) Provision(context.Context, Provisioner) error {
	r.provisions.Add(1)
	return nil
}
func (r *fakeRuntime) Start(context.Context, Host) error { return nil }
func (r *fakeRuntime) Stop(context.Context) error        { return nil }
func (r *fakeRuntime) Load(context.Context, Bundle) (Instance, error) {
	return &fakeInstance{dispatch: func(context.Context, *abi.Event) (abi.Verdict, error) {
		return abi.Verdict{}, nil
	}}, nil
}

func TestRegistryRejectsDuplicateRuntime(t *testing.T) {
	registry := NewRegistry(context.Background(), 0, nil, nil)
	first := &fakeRuntime{name: "test"}
	if err := registry.RegisterRuntime(first); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterRuntime(&fakeRuntime{name: "test"}); err == nil {
		t.Fatal("duplicate runtime was accepted")
	}
	if got, ok := registry.Runtime("test"); !ok || got != first {
		t.Fatalf("Runtime(test) = %v, %v", got, ok)
	}
}

func TestPreflightProvisionsOnlyRequiredRuntime(t *testing.T) {
	registry := NewRegistry(context.Background(), 0, nil, nil)
	needed := &fakeRuntime{name: "needed"}
	unused := &fakeRuntime{name: "unused"}
	for _, runtime := range []*fakeRuntime{needed, unused} {
		if err := registry.RegisterRuntime(runtime); err != nil {
			t.Fatal(err)
		}
	}
	bundles := []Bundle{
		{Bundle: gcpkg.Bundle{Manifest: gcpkg.Manifest{ID: "one", Runtime: "needed"}}},
		{Bundle: gcpkg.Bundle{Manifest: gcpkg.Manifest{ID: "two", Runtime: "needed"}}},
	}
	if err := registry.Preflight(context.Background(), bundles); err != nil {
		t.Fatal(err)
	}
	if needed.provisions.Load() != 1 || unused.provisions.Load() != 0 {
		t.Fatalf("provision counts: needed=%d unused=%d", needed.provisions.Load(), unused.provisions.Load())
	}
}

func TestPreflightReportsMissingRuntime(t *testing.T) {
	registry := NewRegistry(context.Background(), 0, nil, nil)
	err := registry.Preflight(context.Background(), []Bundle{{Bundle: gcpkg.Bundle{Manifest: gcpkg.Manifest{ID: "shop", Runtime: "missing"}}}})
	if err == nil || !strings.Contains(err.Error(), "not available in this build") {
		t.Fatalf("Preflight() error = %v", err)
	}
}
