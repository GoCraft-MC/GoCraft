package player

import (
	"testing"
	"time"
)

func TestRegistryBackedFoodValues(t *testing.T) {
	tests := []struct {
		item       string
		nutrition  int32
		saturation float32
	}{
		{"minecraft:apple", 4, 0.3},
		{"minecraft:rabbit_stew", 10, 0.6},
		{"minecraft:dried_kelp", 1, 0.3},
	}
	for _, test := range tests {
		nutrition, saturation, ok := FoodValue(test.item)
		if !ok || nutrition != test.nutrition || saturation != test.saturation {
			t.Fatalf("FoodValue(%s) = (%d, %v, %v), want (%d, %v, true)",
				test.item, nutrition, saturation, ok, test.nutrition, test.saturation)
		}
	}
	if _, _, ok := FoodValue("minecraft:stone"); ok {
		t.Fatal("stone was classified as food")
	}
}

func TestRegistryBackedConsumptionMetadata(t *testing.T) {
	if got := FoodUseDuration("minecraft:dried_kelp"); got != 800*time.Millisecond {
		t.Fatalf("dried kelp use duration = %v, want 800ms", got)
	}
	if got := FoodUseDuration("minecraft:honey_bottle"); got != 2*time.Second {
		t.Fatalf("honey use duration = %v, want 2s", got)
	}
	if !CanAlwaysEat("minecraft:honey_bottle") || !CanAlwaysEat("minecraft:suspicious_stew") {
		t.Fatal("always-edible component was not preserved")
	}
	if got := FoodRemainder("minecraft:rabbit_stew"); got != "minecraft:bowl" {
		t.Fatalf("rabbit stew remainder = %q, want bowl", got)
	}
}
