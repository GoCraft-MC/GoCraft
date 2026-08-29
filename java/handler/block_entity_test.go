package handler

import (
	"bytes"
	"testing"

	coreworld "GoCraft/core/world"
)

func TestDecoratedPotBlockEntityPacketCarriesSherds(t *testing.T) {
	entity := coreworld.BlockEntity{
		X: 2, Y: 70, Z: 4, Type: "minecraft:decorated_pot",
		PotDecorations: [4]string{
			"minecraft:angler_pottery_sherd", "minecraft:brick",
			"minecraft:flow_pottery_sherd", "minecraft:miner_pottery_sherd",
		},
	}
	pkt := buildBlockEntityData(entity)
	if pkt == nil {
		t.Fatal("decorated pot block entity packet was nil")
	}
	if pkt.ID != packetIDBlockEntityData {
		t.Fatalf("packet ID = %d, want %d", pkt.ID, packetIDBlockEntityData)
	}
	if !bytes.Contains(pkt.Data, []byte("minecraft:flow_pottery_sherd")) {
		t.Fatalf("packet did not contain decorated pot sherd NBT: %x", pkt.Data)
	}
}
