package pluginapi

import (
	"testing"

	abi "GoCraft/abi/v1"
)

type eventPlugin struct {
	joins, breaks int
}

func (p *eventPlugin) OnLoad(context Context) error {
	if err := context.Events().OnPlayerJoin(func(*PlayerJoinEvent) { p.joins++ }); err != nil {
		return err
	}
	return context.Events().OnBlockBreak(func(event *BlockBreakEvent) {
		p.breaks++
		event.Cancel()
	})
}

func (*eventPlugin) OnEnable() error  { return nil }
func (*eventPlugin) OnDisable() error { return nil }

func TestRuntimeDispatchesEventsOnceAndReturnsCancellation(t *testing.T) {
	implementation := &eventPlugin{}
	state := newRuntimeState(Metadata{ID: "events"}, implementation)
	if _, err := state.load("events", "data"); err != nil {
		t.Fatal(err)
	}
	player := abi.List(abi.Bytes(make([]byte, 16)), abi.String("Elias"), abi.String("java"))
	permissions := abi.List(abi.List(abi.String("world.break"), abi.Bool(true)))
	block := abi.List(abi.String("minecraft:stone"), abi.List())
	incoming := &abi.Event{Type: EventBlockBreak, Fields: []abi.Value{
		player,
		abi.List(abi.Int64(1), abi.Int64(64), abi.Int64(2)),
		block,
		abi.String("minecraft:diamond_pickaxe"),
		permissions,
	}}
	verdict, err := state.dispatch(incoming)
	if err != nil {
		t.Fatal(err)
	}
	if !verdict.Cancelled || implementation.breaks != 1 {
		t.Fatalf("verdict cancelled=%v, calls=%d", verdict.Cancelled, implementation.breaks)
	}
	if _, err := state.dispatch(&abi.Event{Type: EventPlayerJoin,
		Fields: []abi.Value{player, permissions}}); err != nil {
		t.Fatal(err)
	}
	if implementation.joins != 1 {
		t.Fatalf("player.join dispatched %d times", implementation.joins)
	}
}

func TestRuntimeRejectsMalformedEvents(t *testing.T) {
	state := newRuntimeState(Metadata{ID: "events"}, &eventPlugin{})
	if _, err := state.load("events", "data"); err != nil {
		t.Fatal(err)
	}
	if _, err := state.dispatch(&abi.Event{Type: EventBlockBreak}); err == nil {
		t.Fatal("malformed block break was accepted")
	}
	state.disable()
	if _, err := state.dispatch(&abi.Event{Type: EventPlayerJoin}); err == nil {
		t.Fatal("disabled plugin received an event")
	}
}
