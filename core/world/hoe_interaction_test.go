package world

import "testing"

func TestUseHoeTransformsSoilAndReturnsRoots(t *testing.T) {
	farmland, drop, ok := UseHoe(Block{Namespace: "minecraft", Name: "grass_block"}, true)
	if !ok || farmland.ResourceLocation() != "minecraft:farmland" || farmland.Properties["moisture"] != "0" || !drop.IsEmpty() {
		t.Fatalf("grass hoe result = %+v, drop=%+v, ok=%v", farmland, drop, ok)
	}
	if _, _, ok := UseHoe(Block{Namespace: "minecraft", Name: "dirt"}, false); ok {
		t.Fatal("dirt became farmland without a valid face and clear space")
	}
	dirt, roots, ok := UseHoe(Block{Namespace: "minecraft", Name: "rooted_dirt"}, false)
	if !ok || dirt.ResourceLocation() != "minecraft:dirt" || roots.ItemID != "minecraft:hanging_roots" || roots.Count != 1 {
		t.Fatalf("rooted dirt hoe result = %+v, drop=%+v, ok=%v", dirt, roots, ok)
	}
	if _, _, ok := UseHoe(Block{Namespace: "minecraft", Name: "stone"}, true); ok {
		t.Fatal("stone accepted a hoe transformation")
	}
}
