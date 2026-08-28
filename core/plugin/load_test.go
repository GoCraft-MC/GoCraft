package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"

	abi "GoCraft/abi/v1"
)

type recordingInstance struct {
	manifest Manifest
	order    *[]string
}

func (i *recordingInstance) Manifest() Manifest { return i.manifest }
func (i *recordingInstance) Dispatch(context.Context, *abi.Event) (abi.Verdict, error) {
	return abi.Verdict{}, nil
}
func (i *recordingInstance) Unload(context.Context) error {
	*i.order = append(*i.order, "unload:"+i.manifest.ID)
	return nil
}

type recordingRuntime struct {
	order  *[]string
	failID string
}

func (r *recordingRuntime) Name() string                                 { return "recording" }
func (r *recordingRuntime) Provision(context.Context, Provisioner) error { return nil }
func (r *recordingRuntime) Start(context.Context, Host) error {
	*r.order = append(*r.order, "start")
	return nil
}
func (r *recordingRuntime) Load(_ context.Context, bundle Bundle) (Instance, error) {
	*r.order = append(*r.order, "load:"+bundle.Manifest.ID)
	if bundle.Manifest.ID == r.failID {
		return nil, errors.New("load failed")
	}
	return &recordingInstance{manifest: bundle.Manifest, order: r.order}, nil
}
func (r *recordingRuntime) Stop(context.Context) error {
	*r.order = append(*r.order, "stop")
	return nil
}

func testBundle(id string) Bundle {
	return Bundle{Manifest: Manifest{ID: id, Version: "1.0.0", APIVersion: 1, Runtime: "recording"}}
}

func TestLoadAllAndStopUseDeterministicReverseOrder(t *testing.T) {
	var order []string
	registry := NewRegistry(context.Background(), 0, nil, nil)
	if err := registry.RegisterRuntime(&recordingRuntime{order: &order}); err != nil {
		t.Fatal(err)
	}
	if err := registry.LoadAll(context.Background(), []Bundle{testBundle("b"), testBundle("a")}); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Instance("a"); !ok {
		t.Fatal("loaded instance is missing")
	}
	if err := registry.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"start", "load:a", "load:b", "unload:b", "unload:a", "stop"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("lifecycle order = %v, want %v", order, want)
	}
}

func TestLoadAllRollsBackPartialStartup(t *testing.T) {
	var order []string
	registry := NewRegistry(context.Background(), 0, nil, nil)
	runtime := &recordingRuntime{order: &order, failID: "b"}
	if err := registry.RegisterRuntime(runtime); err != nil {
		t.Fatal(err)
	}
	err := registry.LoadAll(context.Background(), []Bundle{testBundle("a"), testBundle("b")})
	if err == nil {
		t.Fatal("failed runtime load was accepted")
	}
	want := []string{"start", "load:a", "load:b", "unload:a", "stop"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("rollback order = %v, want %v", order, want)
	}
	if _, ok := registry.Instance("a"); ok {
		t.Fatal("rolled-back instance remains registered")
	}
	if _, ok := registry.Bus().Health("a"); ok {
		t.Fatal("rolled-back event subscriptions remain attached")
	}
}
