package pluginapi

import "testing"

func TestBlockBreakEventCancellation(t *testing.T) {
	event := &BlockBreakEvent{
		Player:   Player{Username: "Elias"},
		Position: BlockPos{X: 4, Y: 64, Z: -2},
		Block:    Block{ID: "minecraft:stone"},
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
