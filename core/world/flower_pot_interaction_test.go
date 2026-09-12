package world

import "testing"

func TestFlowerPotMappingsRoundTrip(t *testing.T) {
	for _, item := range []string{"minecraft:poppy", "minecraft:cactus", "minecraft:flowering_azalea"} {
		block, ok := PottedBlock(item)
		if !ok {
			t.Fatalf("PottedBlock rejected %s", item)
		}
		if got, ok := PottedItem(block); !ok || got != item {
			t.Errorf("PottedItem(%s) = %q, %v", block.ResourceLocation(), got, ok)
		}
	}
	if _, ok := PottedBlock("minecraft:stone"); ok {
		t.Fatal("flower pot accepted stone")
	}
	if _, ok := PottedItem(Block{Namespace: "minecraft", Name: "potted_stone"}); ok {
		t.Fatal("unknown potted block produced an item")
	}
}
