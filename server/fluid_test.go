package server

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestFluidSpreadRulesMatchDimension(t *testing.T) {
	tests := []struct {
		name      string
		dimension int32
		level     int
		delay     int64
	}{
		{"minecraft:water", dimensionOverworld, 7, 5},
		{"minecraft:water", dimensionNether, 7, 5},
		{"minecraft:lava", dimensionOverworld, 3, 30},
		{"minecraft:lava", dimensionEnd, 3, 30},
		{"minecraft:lava", dimensionNether, 7, 10},
	}
	for _, test := range tests {
		level, delay := fluidSpreadRules(test.name, test.dimension)
		if level != test.level || delay != test.delay {
			t.Errorf("%s dimension %d = (%d, %d), want (%d, %d)",
				test.name, test.dimension, level, delay, test.level, test.delay)
		}
	}
}

func TestFluidCollisionsMatchVanilla(t *testing.T) {
	tests := []struct {
		name       string
		fluid      coreworld.Block
		waterBelow bool
		want       string
	}{
		{"source lava beside water", coreworld.MakeFluid("minecraft:lava", 0), false, "minecraft:obsidian"},
		{"flowing lava beside water", coreworld.MakeFluid("minecraft:lava", 2), false, "minecraft:cobblestone"},
		{"falling lava enters water", coreworld.MakeFluid("minecraft:lava", 0), true, "minecraft:stone"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer w.Close()
			w.SetBlock(0, 65, 0, test.fluid)
			waterX, waterY := 1, 65
			if test.waterBelow {
				waterX, waterY = 0, 64
			}
			w.SetBlock(waterX, waterY, 0, coreworld.MakeFluid("minecraft:water", 0))
			s := &Server{world: w, simulationDimension: dimensionOverworld}
			var changes []coreworld.BlockChange
			s.processFluidUpdate(0, 65, 0, &changes)
			got := w.GetBlock(0, 65, 0).ResourceLocation()
			if test.waterBelow {
				got = w.GetBlock(0, 64, 0).ResourceLocation()
			}
			if got != test.want {
				t.Fatalf("collision result = %s, want %s", got, test.want)
			}
		})
	}
}

func TestWaterHardensExistingLava(t *testing.T) {
	for level, want := range map[int]string{0: "minecraft:obsidian", 2: "minecraft:cobblestone"} {
		w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
		w.SetBlock(0, 65, 0, coreworld.MakeFluid("minecraft:water", 0))
		w.SetBlock(1, 65, 0, coreworld.MakeFluid("minecraft:lava", level))
		s := &Server{world: w, simulationDimension: dimensionOverworld}
		var changes []coreworld.BlockChange
		s.processFluidUpdate(0, 65, 0, &changes)
		if got := w.GetBlock(1, 65, 0).ResourceLocation(); got != want {
			t.Errorf("lava level %d became %s, want %s", level, got, want)
		}
		w.Close()
	}
}
