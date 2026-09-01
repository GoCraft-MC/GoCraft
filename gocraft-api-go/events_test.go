package gocraft

import (
	"testing"

	abi "GoCraft/abi/v1"
)

func TestBlockBreakEventCancellation(t *testing.T) {
	event := &BlockBreakEvent{
		Player: Player{Username: "Elias"},
		Pos:    BlockPos{X: 4, Y: 64, Z: -2},
		Block:  Block{ID: "minecraft:stone"},
	}
	if event.Type() != EventBlockBreak {
		t.Fatalf("Type() = %q", event.Type())
	}
	if event.Cancelled() {
		t.Fatal("new event is cancelled")
	}
	event.Cancel()
	if !event.Cancelled() {
		t.Fatal("Cancel() did not persist")
	}
}

// An observational event offers no way to cancel it, because the tick never
// waits for one and a Cancel that did nothing would be worse than its absence.
func TestPlayerJoinEventIsNotCancellable(t *testing.T) {
	var event Event = &PlayerJoinEvent{Player: Player{Username: "Elias"}}
	if _, cancellable := event.(CancellableEvent); cancellable {
		t.Fatal("an observational event offered cancellation")
	}
}

// Decoding is what proves the layout: the payload carries no field names, so a
// reader that disagreed with the host about index 3 would produce a plausible
// event rather than an error.
func TestBlockBreakDecodesItsPositionalPayload(t *testing.T) {
	uuid := make([]byte, 16)
	uuid[0] = 7
	fields := []abi.Value{
		abi.List(abi.Bytes(uuid), abi.String("Elias"), abi.String("bedrock")),
		abi.List(abi.Int64(4), abi.Int64(64), abi.Int64(-2)),
		abi.List(abi.String("minecraft:stone"), abi.List()),
		abi.String("minecraft:diamond_pickaxe"),
		abi.List(abi.List(abi.String("spawn.bypass"), abi.Bool(true))),
	}
	decoded, err := eventFrom(&abi.Event{Type: EventBlockBreak, Fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	event, ok := decoded.(*BlockBreakEvent)
	if !ok {
		t.Fatalf("decoded %T", decoded)
	}
	if event.Player.Username != "Elias" || event.Player.Edition != "bedrock" {
		t.Fatalf("player = %+v", event.Player)
	}
	if event.Pos != (BlockPos{X: 4, Y: 64, Z: -2}) {
		t.Fatalf("pos = %+v", event.Pos)
	}
	if event.Block.ID != "minecraft:stone" {
		t.Fatalf("block = %+v", event.Block)
	}
	if event.Tool != "minecraft:diamond_pickaxe" {
		t.Fatalf("tool = %q", event.Tool)
	}
	// The injected map is answered through a query, never handed over.
	if !event.Can("spawn.bypass") || event.Can("never.declared") {
		t.Fatal("resolved permissions did not arrive")
	}
}

// A coordinate the schema declares as sint64 has to survive as one. Narrowing
// it would be a silent truncation nothing in the build would catch.
func TestBlockPositionKeepsSixtyFourBits(t *testing.T) {
	const far = int64(1) << 40
	position, err := positionFrom(abi.List(abi.Int64(far), abi.Int64(-far), abi.Int64(far)))
	if err != nil {
		t.Fatal(err)
	}
	if position.X != far || position.Y != -far || position.Z != far {
		t.Fatalf("position = %+v", position)
	}
}

func TestEventFromRefusesWhatItCannotRead(t *testing.T) {
	if _, err := eventFrom(nil); err == nil {
		t.Fatal("a missing event decoded")
	}
	if _, err := eventFrom(&abi.Event{Type: "shop.purchase"}); err == nil {
		t.Fatal("an unknown event decoded")
	}
	if _, err := eventFrom(&abi.Event{Type: EventPlayerJoin}); err == nil {
		t.Fatal("a payload of the wrong length decoded")
	}
}
