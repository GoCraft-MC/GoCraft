package player

import "testing"

func TestFoodStatusEffects(t *testing.T) {
	if effects := FoodStatusEffects("minecraft:rotten_flesh", 79); len(effects) != 1 || effects[0].ID != "hunger" {
		t.Fatalf("rotten flesh effects = %#v", effects)
	}
	if effects := FoodStatusEffects("minecraft:rotten_flesh", 80); len(effects) != 0 {
		t.Fatalf("failed probability roll produced %#v", effects)
	}
	if effects := FoodStatusEffects("minecraft:pufferfish", 99); len(effects) != 3 || effects[0].Amplifier != 3 {
		t.Fatalf("pufferfish effects = %#v", effects)
	}
	if effects := FoodStatusEffects("minecraft:enchanted_golden_apple", 0); len(effects) != 4 || effects[1].ID != "absorption" || effects[1].Amplifier != 3 {
		t.Fatalf("enchanted golden apple effects = %#v", effects)
	}
}
