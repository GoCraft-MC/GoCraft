package world

import (
	"strconv"
	"testing"
)

func tallPlant(name string, age int) Block {
	return Block{Namespace: "minecraft", Name: name, Properties: map[string]string{"age": strconv.Itoa(age)}}
}

func TestSugarCaneAgeIncrementsBelowMax(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "sand"})
	w.SetBlock(0, 41, 0, tallPlant("sugar_cane", 0))
	w.TickCrops(0, 4)
	if got := tallPlantAge(w.GetBlock(0, 41, 0)); got != 1 {
		t.Fatalf("sugar cane age = %d, want 1", got)
	}
	if !w.GetBlock(0, 42, 0).IsAir() {
		t.Fatal("sugar cane grew a segment before reaching age 15")
	}
}

func TestSugarCaneGrowsSegmentAtMaxAge(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "sand"})
	w.SetBlock(0, 41, 0, tallPlant("sugar_cane", 15))
	w.TickCrops(0, 4)
	top := w.GetBlock(0, 42, 0)
	if top.ResourceLocation() != "minecraft:sugar_cane" || tallPlantAge(top) != 0 {
		t.Fatalf("grown top = %s age %d, want fresh sugar cane", top.ResourceLocation(), tallPlantAge(top))
	}
	if got := tallPlantAge(w.GetBlock(0, 41, 0)); got != 0 {
		t.Fatalf("base age after growth = %d, want reset to 0", got)
	}
}

func TestSugarCaneStopsAtHeightThree(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "sand"})
	w.SetBlock(0, 41, 0, tallPlant("sugar_cane", 0))
	w.SetBlock(0, 42, 0, tallPlant("sugar_cane", 0))
	w.SetBlock(0, 43, 0, tallPlant("sugar_cane", 15))
	w.TickCrops(0, 4)
	if !w.GetBlock(0, 44, 0).IsAir() {
		t.Fatal("sugar cane grew beyond the three-block height limit")
	}
}

func TestCactusGrowsWhenSidesAreClear(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "sand"})
	w.SetBlock(0, 41, 0, tallPlant("cactus", 15))
	w.TickCrops(0, 4)
	if got := w.GetBlock(0, 42, 0).ResourceLocation(); got != "minecraft:cactus" {
		t.Fatalf("cactus above = %s, want minecraft:cactus", got)
	}
}

func TestCactusBlockedBySolidNeighbour(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "sand"})
	w.SetBlock(0, 41, 0, tallPlant("cactus", 15))
	w.SetBlock(1, 42, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.TickCrops(0, 4)
	if !w.GetBlock(0, 42, 0).IsAir() {
		t.Fatal("cactus grew next to a solid block, violating its survival rule")
	}
	if got := tallPlantAge(w.GetBlock(0, 41, 0)); got != 15 {
		t.Fatalf("blocked cactus base age = %d, want 15 (retry later)", got)
	}
}
