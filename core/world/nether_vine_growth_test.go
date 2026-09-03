package world

import (
	"strconv"
	"testing"
)

func netherVine(name string, age int) Block {
	return Block{Namespace: "minecraft", Name: name, Properties: map[string]string{"age": strconv.Itoa(age)}}
}

func netherVineGrowTick(t *testing.T, y int) int64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		if cropRandom(seed, 0, y, 0, netherVineGrowthSalt, 10) == 0 {
			return int64(seed * 20)
		}
	}
	t.Fatal("no nether vine growth tick found")
	return 0
}

func TestTwistingVinesGrowUpward(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 100, 0, netherVine("twisting_vines", 0))
	w.TickCrops(netherVineGrowTick(t, 100), 4)

	tip := w.GetBlock(0, 101, 0)
	if tip.ResourceLocation() != "minecraft:twisting_vines" || kelpAge(tip) != 1 {
		t.Fatalf("new tip = %s age %d, want twisting_vines age 1", tip.ResourceLocation(), kelpAge(tip))
	}
	if got := w.GetBlock(0, 100, 0).ResourceLocation(); got != "minecraft:twisting_vines_plant" {
		t.Fatalf("old tip = %s, want twisting_vines_plant", got)
	}
}

func TestWeepingVinesGrowDownward(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 100, 0, netherVine("weeping_vines", 0))
	w.TickCrops(netherVineGrowTick(t, 100), 4)

	tip := w.GetBlock(0, 99, 0)
	if tip.ResourceLocation() != "minecraft:weeping_vines" || kelpAge(tip) != 1 {
		t.Fatalf("new tip = %s age %d, want weeping_vines age 1", tip.ResourceLocation(), kelpAge(tip))
	}
	if got := w.GetBlock(0, 100, 0).ResourceLocation(); got != "minecraft:weeping_vines_plant" {
		t.Fatalf("old tip = %s, want weeping_vines_plant", got)
	}
}

func TestNetherVineStopsAtMaxAge(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 100, 0, netherVine("twisting_vines", kelpMaxAge))
	w.TickCrops(netherVineGrowTick(t, 100), 4)
	if !w.GetBlock(0, 101, 0).IsAir() {
		t.Fatal("fully grown twisting vine advanced past age 25")
	}
}
