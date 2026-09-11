package handler

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/protocol"
)

func TestLecternButtonsPostPageIntents(t *testing.T) {
	tests := []struct {
		button   int32
		page     int
		relative bool
	}{{1, -1, true}, {2, 1, true}, {107, 7, false}}
	for _, test := range tests {
		bus := intent.NewBus(1, 1)
		p := player.New([16]byte{7}, "reader", player.ClientEditionJava)
		p.Dimension = 2
		p.OpenContainerID = chestContainerID
		p.OpenContainerKind = "minecraft:lectern"
		p.OpenContainerPos = spatial.BlockPos{X: 4, Y: 70, Z: -3}
		pkt := protocol.NewBuilder(packetIDContainerButtonClick).
			VarInt(chestContainerID).VarInt(test.button).Build()
		if err := handleLecternButtonClick(pkt, p, bus); err != nil {
			t.Fatalf("button %d: %v", test.button, err)
		}
		queued := bus.Drain().Gameplay
		if len(queued) != 1 {
			t.Fatalf("button %d queued %d intents, want 1", test.button, len(queued))
		}
		got := queued[0].(intent.LecternPageIntent)
		if got.Page != test.page || got.Relative != test.relative || got.Position != p.OpenContainerPos || got.Dimension != 2 {
			t.Fatalf("button %d intent = %+v", test.button, got)
		}
	}
}

func TestLecternButtonIgnoresClosedScreen(t *testing.T) {
	bus := intent.NewBus(1, 1)
	p := player.New([16]byte{7}, "reader", player.ClientEditionJava)
	pkt := protocol.NewBuilder(packetIDContainerButtonClick).VarInt(chestContainerID).VarInt(2).Build()
	if err := handleLecternButtonClick(pkt, p, bus); err != nil {
		t.Fatal(err)
	}
	if got := len(bus.Drain().Gameplay); got != 0 {
		t.Fatalf("closed lectern queued %d intents", got)
	}
}
