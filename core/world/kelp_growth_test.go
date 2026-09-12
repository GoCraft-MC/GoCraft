package world

import (
	"strconv"
	"testing"
)

func kelpTip(age int) Block {
	return Block{Namespace: "minecraft", Name: "kelp", Properties: map[string]string{"age": strconv.Itoa(age)}}
}

// kelpGrowTick returns a tick whose per-second seed passes the kelp growth gate
// at the origin column.
func kelpGrowTick(t *testing.T) int64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		if cropRandom(seed, 0, 64, 0, kelpGrowthSalt, 7) == 0 {
			return int64(seed * 20)
		}
	}
	t.Fatal("no kelp growth tick found")
	return 0
}

func TestKelpGrowsUpwardIntoWater(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 63, 0, Block{Namespace: "minecraft", Name: "dirt"})
	w.SetBlock(0, 64, 0, kelpTip(0))
	w.SetBlock(0, 65, 0, MakeFluid("minecraft:water", 0))
	w.TickCrops(kelpGrowTick(t), 4)

	tip := w.GetBlock(0, 65, 0)
	if tip.ResourceLocation() != "minecraft:kelp" || kelpAge(tip) != 1 {
		t.Fatalf("new tip = %s age %d, want kelp age 1", tip.ResourceLocation(), kelpAge(tip))
	}
	if got := w.GetBlock(0, 64, 0).ResourceLocation(); got != "minecraft:kelp_plant" {
		t.Fatalf("old tip = %s, want minecraft:kelp_plant", got)
	}
}

func TestKelpStopsAtMaxAge(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, kelpTip(kelpMaxAge))
	w.SetBlock(0, 65, 0, MakeFluid("minecraft:water", 0))
	w.TickCrops(kelpGrowTick(t), 4)
	if got := w.GetBlock(0, 65, 0).ResourceLocation(); got != "minecraft:water" {
		t.Fatalf("fully grown kelp advanced into %s, want water untouched", got)
	}
}

func TestKelpDoesNotGrowWithoutWaterAbove(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, kelpTip(0))
	// Air above (no water column) must block growth.
	w.TickCrops(kelpGrowTick(t), 4)
	if !w.GetBlock(0, 65, 0).IsAir() {
		t.Fatal("kelp grew into air above the column")
	}
	if got := kelpAge(w.GetBlock(0, 64, 0)); got != 0 {
		t.Fatalf("blocked kelp tip age = %d, want 0", got)
	}
}
