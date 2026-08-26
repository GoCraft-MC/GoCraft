package server

import "testing"

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
