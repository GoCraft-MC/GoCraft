package pluginapi

import (
	"net"
	"testing"

	abi "GoCraft/abi/v1"
	wire "GoCraft/abi/v1/wire"
	"GoCraft/runtime/ipc"
)

type sessionPlugin struct {
	entered  chan struct{}
	release  chan struct{}
	disabled int
}

func (p *sessionPlugin) OnLoad(context Context) error {
	return context.Events().OnBlockBreak(func(event *BlockBreakEvent) {
		close(p.entered)
		<-p.release
		event.Cancel()
	})
}

func (*sessionPlugin) OnEnable() error { return nil }
func (p *sessionPlugin) OnDisable() error {
	p.disabled++
	return nil
}

func TestSessionKeepsHeartbeatResponsiveDuringEvent(t *testing.T) {
	hostStream, childStream := net.Pipe()
	host := ipc.NewCodec(hostStream)
	implementation := &sessionPlugin{entered: make(chan struct{}), release: make(chan struct{})}
	state := newRuntimeState(Metadata{ID: "example"}, implementation)
	result := make(chan error, 1)
	go func() { result <- serve(ipc.NewCodec(childStream), state) }()

	if err := host.Send(&wire.Envelope{Seq: 1, Body: &wire.Envelope_Load{Load: &wire.Load{
		PluginId: "example", DataDirectory: "data",
	}}}); err != nil {
		t.Fatal(err)
	}
	loaded, err := host.Receive()
	if err != nil || loaded.GetLoaded() == nil || len(loaded.GetLoaded().GetEvents()) != 1 {
		t.Fatalf("LOAD response = %#v, %v", loaded, err)
	}
	event, err := ipc.EncodeEvent(testBlockBreakABI())
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Send(&wire.Envelope{Seq: 2, Body: &wire.Envelope_Dispatch{Dispatch: &wire.Dispatch{
		PluginId: "example", Event: event,
	}}}); err != nil {
		t.Fatal(err)
	}
	<-implementation.entered
	if err := host.Send(&wire.Envelope{Seq: 3, Body: &wire.Envelope_Ping{Ping: &wire.Ping{}}}); err != nil {
		t.Fatal(err)
	}
	pong, err := host.Receive()
	if err != nil || pong.GetSeq() != 3 || pong.GetPong() == nil {
		t.Fatalf("PING response = %#v, %v", pong, err)
	}
	close(implementation.release)
	verdict, err := host.Receive()
	if err != nil || verdict.GetSeq() != 2 || !verdict.GetVerdict().GetCancelled() {
		t.Fatalf("DISPATCH response = %#v, %v", verdict, err)
	}
	if err := host.Send(&wire.Envelope{Body: &wire.Envelope_Shutdown{Shutdown: &wire.Shutdown{}}}); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err != nil || implementation.disabled != 1 {
		t.Fatalf("serve() = %v, disabled=%d", err, implementation.disabled)
	}
}

func testBlockBreakABI() *abi.Event {
	player := abi.List(abi.Bytes(make([]byte, 16)), abi.String("Elias"), abi.String("java"))
	return &abi.Event{Type: EventBlockBreak, OnFailure: abi.FailureAllow, Fields: []abi.Value{
		player, abi.List(abi.Int64(1), abi.Int64(64), abi.Int64(2)),
		abi.List(abi.String("minecraft:stone"), abi.List()), abi.String("minecraft:pickaxe"), abi.List(),
	}}
}
