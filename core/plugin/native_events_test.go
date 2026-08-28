package plugin

import (
	"context"
	"testing"
	"time"

	abi "GoCraft/abi/v1"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestBlockBreakPayloadIsEditionNeutral(t *testing.T) {
	bus := NewBus(context.Background(), time.Second)
	var received *abi.Event
	instance := &fakeInstance{
		manifest: Manifest{ID: "protect", Subscriptions: []Subscription{{Event: EventBlockBreak}}},
		dispatch: func(_ context.Context, event *abi.Event) (abi.Verdict, error) {
			received = event
			return abi.Verdict{Cancelled: true}, nil
		},
	}
	if err := bus.Attach(instance); err != nil {
		t.Fatal(err)
	}
	identity := [16]byte{1, 2, 3}
	p := player.New(identity, "Alex", player.ClientEditionBedrock)
	block := coreworld.Block{Namespace: "minecraft", Name: "oak_log", Properties: map[string]string{
		"waterlogged": "false", "axis": "y",
	}}
	allowed := bus.EmitBlockBreak(p, spatial.BlockPos{X: 4, Y: 64, Z: -2}, block,
		player.ItemStack{ItemID: "minecraft:iron_axe"})
	if allowed || received == nil {
		t.Fatalf("allowed = %v, event = %v", allowed, received)
	}
	if received.Type != EventBlockBreak || received.OnFailure != abi.FailureAllow {
		t.Fatalf("event metadata = %+v", received)
	}
	playerFields := received.Fields[0].List
	if string(playerFields[0].Bytes) != string(identity[:]) || playerFields[1].String != "Alex" || playerFields[2].String != "bedrock" {
		t.Fatalf("player reference = %+v", playerFields)
	}
	position := received.Fields[1].List
	if position[0].Int64 != 4 || position[1].Int64 != 64 || position[2].Int64 != -2 {
		t.Fatalf("position = %+v", position)
	}
	blockFields := received.Fields[2].List
	if blockFields[0].String != "minecraft:oak_log" {
		t.Fatalf("block name = %q", blockFields[0].String)
	}
	properties := blockFields[1].List
	if properties[0].List[0].String != "axis" || properties[1].List[0].String != "waterlogged" {
		t.Fatalf("properties not sorted: %+v", properties)
	}
	if received.Fields[3].String != "minecraft:iron_axe" {
		t.Fatalf("tool = %q", received.Fields[3].String)
	}
}
