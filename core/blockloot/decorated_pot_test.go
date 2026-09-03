package blockloot

import (
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func TestDecoratedPotIntactDropPreservesDecorations(t *testing.T) {
	decorations := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:heart_pottery_sherd"}
	drops := Drops(Context{
		Block:          coreworld.Block{Namespace: "minecraft", Name: "decorated_pot", Properties: map[string]string{"cracked": "false"}},
		PotDecorations: decorations,
	})
	if len(drops) != 1 || drops[0].ItemID != "minecraft:decorated_pot" || drops[0].Count != 1 {
		t.Fatalf("intact pot drops = %#v", drops)
	}
	if got := drops[0].NormalizedPotDecorations(); got != decorations {
		t.Fatalf("intact pot decorations = %#v, want %#v", got, decorations)
	}
}

func TestDecoratedPotShatterReturnsOriginalSherds(t *testing.T) {
	decorations := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:angler_pottery_sherd"}
	drops := Drops(Context{
		Block:          coreworld.Block{Namespace: "minecraft", Name: "decorated_pot", Properties: map[string]string{"cracked": "true"}},
		Tool:           player.ItemStack{ItemID: "minecraft:iron_pickaxe", Count: 1},
		PotDecorations: decorations,
	})
	counts := map[string]int{}
	for _, drop := range drops {
		counts[drop.ItemID] += drop.Count
	}
	want := map[string]int{"minecraft:angler_pottery_sherd": 2, "minecraft:brick": 1, "minecraft:skull_pottery_sherd": 1}
	if len(counts) != len(want) {
		t.Fatalf("shattered drops = %#v, want %#v", counts, want)
	}
	for item, count := range want {
		if counts[item] != count {
			t.Fatalf("%s count = %d, want %d (all=%#v)", item, counts[item], count, counts)
		}
	}
}
