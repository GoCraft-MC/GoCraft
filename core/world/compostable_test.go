package world

import "testing"

func TestCompostableClassificationAndChance(t *testing.T) {
	for _, item := range []string{"minecraft:oak_leaves", "minecraft:wheat_seeds", "minecraft:bread", "minecraft:azalea"} {
		if !IsCompostable(item) {
			t.Errorf("%s is not compostable", item)
		}
	}
	if IsCompostable("minecraft:stone") {
		t.Fatal("stone is compostable")
	}
	for tick := int64(0); tick < 20; tick++ {
		if !ComposterAccepts(1, 64, 2, tick, "minecraft:cake") {
			t.Fatalf("cake failed a guaranteed compost roll at tick %d", tick)
		}
	}
}
