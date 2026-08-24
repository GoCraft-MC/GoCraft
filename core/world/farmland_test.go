package world

import "testing"

func TestFarmlandHydratesFromNearbyWater(t *testing.T) {
	world := New(&FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "farmland", Properties: map[string]string{"moisture": "0"}})
	world.SetBlock(4, 40, 0, Block{Namespace: "minecraft", Name: "water", Properties: map[string]string{"level": "0"}})
	for tick := int64(20); tick <= 400; tick += 20 {
		world.TickFarmland(tick, 64)
		if world.GetBlock(0, 40, 0).Properties["moisture"] == "7" {
			return
		}
	}
	t.Fatal("farmland did not hydrate from water four blocks away")
}

func TestDryEmptyFarmlandReturnsToDirt(t *testing.T) {
	world := New(&FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "farmland", Properties: map[string]string{"moisture": "0"}})
	for tick := int64(20); tick <= 400; tick += 20 {
		world.TickFarmland(tick, 64)
		if world.GetBlock(0, 40, 0).ResourceLocation() == "minecraft:dirt" {
			return
		}
	}
	t.Fatal("dry empty farmland did not return to dirt")
}

func TestDryFarmlandRemainsUnderValidCrop(t *testing.T) {
	world := New(&FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "farmland", Properties: map[string]string{"moisture": "0"}})
	world.SetBlock(0, 41, 0, Block{Namespace: "minecraft", Name: "wheat", Properties: map[string]string{"age": "7"}})
	for tick := int64(20); tick <= 1000; tick += 20 {
		world.TickFarmland(tick, 64)
	}
	if got := world.GetBlock(0, 40, 0).ResourceLocation(); got != "minecraft:farmland" {
		t.Fatalf("farmland under wheat became %q", got)
	}
}

func TestFarmlandWithSolidBlockAboveReturnsToDirt(t *testing.T) {
	world := New(&FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "farmland", Properties: map[string]string{"moisture": "7"}})
	world.SetBlock(0, 41, 0, Block{Namespace: "minecraft", Name: "stone"})
	for tick := int64(20); tick <= 1000; tick += 20 {
		world.TickFarmland(tick, 64)
		if world.GetBlock(0, 40, 0).ResourceLocation() == "minecraft:dirt" {
			return
		}
	}
	t.Fatal("farmland with a solid block above did not become dirt")
}
