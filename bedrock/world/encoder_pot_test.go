package world

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestDecoratedPotBlockActorDataUsesCanonicalSherds(t *testing.T) {
	entity := coreworld.BlockEntity{
		X: 7, Y: 65, Z: -4, Type: "minecraft:decorated_pot",
		PotDecorations: [4]string{
			"minecraft:angler_pottery_sherd", "minecraft:brick",
			"minecraft:flow_pottery_sherd", "minecraft:miner_pottery_sherd",
		},
	}
	data, ok := bedrockBlockEntityData(entity)
	if !ok {
		t.Fatal("decorated pot was not encoded as a Bedrock block actor")
	}
	if data["id"] != "DecoratedPot" || data["x"] != int32(7) || data["y"] != int32(65) || data["z"] != int32(-4) {
		t.Fatalf("unexpected block actor identity/position: %#v", data)
	}
	sherds, ok := data["sherds"].([]string)
	if !ok || len(sherds) != 4 {
		t.Fatalf("unexpected sherd payload: %#v", data["sherds"])
	}
	for index, want := range entity.PotDecorations {
		if sherds[index] != want {
			t.Fatalf("sherd %d = %q, want %q", index, sherds[index], want)
		}
	}
}
