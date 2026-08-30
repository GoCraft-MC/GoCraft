package plugin

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	abi "GoCraft/abi/v1"
	"GoCraft/core/command"
)

type recordingInstance struct {
	manifest Manifest
	order    *[]string
}

func (i *recordingInstance) Manifest() Manifest { return i.manifest }
func (i *recordingInstance) Dispatch(context.Context, *abi.Event) (abi.Verdict, error) {
	return abi.Verdict{}, nil
}
func (i *recordingInstance) InvokeCommand(_ context.Context, executor command.ExecID, _ command.Sender, _ command.Values) error {
	*i.order = append(*i.order, fmt.Sprintf("command:%d", executor))
	return nil
}
func (i *recordingInstance) Unload(context.Context) error {
	*i.order = append(*i.order, "unload:"+i.manifest.ID)
	return nil
}

type recordingRuntime struct {
	order   *[]string
	failID  string
	onStart func()
	loaded  *[]Bundle
}

type noCommandRuntime struct{ recordingRuntime }

func (r *noCommandRuntime) Load(_ context.Context, bundle Bundle) (Instance, error) {
	*r.order = append(*r.order, "load:"+bundle.Manifest.ID)
	return &fakeInstance{manifest: bundle.Manifest}, nil
}

func (r *recordingRuntime) Name() string                                 { return "recording" }
func (r *recordingRuntime) Provision(context.Context, Provisioner) error { return nil }
func (r *recordingRuntime) Start(context.Context, Host) error {
	*r.order = append(*r.order, "start")
	if r.onStart != nil {
		r.onStart()
	}
	return nil
}

func TestLoadAllRegistersAndRevokesPluginCommands(t *testing.T) {
	var order []string
	registry := NewRegistry(context.Background(), 0, nil, nil)
	runtime := &recordingRuntime{order: &order, onStart: func() {
		if got := registry.Commands().Snapshot(nil).Root.Children; len(got) != 1 {
			t.Fatalf("commands at runtime start = %#v", got)
		}
	}}
	if err := registry.RegisterRuntime(runtime); err != nil {
		t.Fatal(err)
	}
	bundle := testBundle("shop")
	tree := command.Root{Children: []command.Node{command.Literal{Name: "shop", Exec: 7}}}
	bundle.Commands = &tree
	if err := registry.LoadAll(context.Background(), []Bundle{bundle}); err != nil {
		t.Fatal(err)
	}
	snapshot := registry.Commands().Snapshot(nil)
	executor := snapshot.Root.Children[0].(command.Literal).Exec
	if err := registry.Commands().Invoke(context.Background(), executor, nil, nil); err != nil {
		t.Fatal(err)
	}
	if order[len(order)-1] != "command:7" {
		t.Fatalf("command callback order = %v", order)
	}
	if err := registry.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := registry.Commands().Snapshot(nil).Root.Children; len(got) != 0 {
		t.Fatalf("commands remain after stop: %#v", got)
	}
}

func TestLoadAllRejectsRuntimeWithoutCommandSupport(t *testing.T) {
	var order []string
	registry := NewRegistry(context.Background(), 0, nil, nil)
	if err := registry.RegisterRuntime(&noCommandRuntime{recordingRuntime{order: &order}}); err != nil {
		t.Fatal(err)
	}
	bundle := testBundle("shop")
	tree := command.Root{Children: []command.Node{command.Literal{Name: "shop", Exec: 1}}}
	bundle.Commands = &tree
	err := registry.LoadAll(context.Background(), []Bundle{bundle})
	if err == nil || !strings.Contains(err.Error(), "does not support commands") {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if got := registry.Commands().Snapshot(nil).Root.Children; len(got) != 0 {
		t.Fatalf("failed load left commands: %#v", got)
	}
}
func (r *recordingRuntime) Load(_ context.Context, bundle Bundle) (Instance, error) {
	*r.order = append(*r.order, "load:"+bundle.Manifest.ID)
	if r.loaded != nil {
		*r.loaded = append(*r.loaded, bundle)
	}
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
