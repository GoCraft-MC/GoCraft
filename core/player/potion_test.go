package player

import "testing"

func TestPotionOutcomeForBasePotions(t *testing.T) {
	tests := []struct {
		potion       string
		effect       string
		amplifier    int32
		duration     int32
		heal, damage float32
	}{
		{potion: "long_swiftness", effect: "minecraft:speed", duration: 9600},
		{potion: "strong_leaping", effect: "minecraft:jump_boost", amplifier: 1, duration: 1800},
		{potion: "strong_poison", effect: "minecraft:poison", amplifier: 1, duration: 432},
		{potion: "strong_slowness", effect: "minecraft:slowness", amplifier: 3, duration: 400},
		{potion: "healing", heal: 4},
		{potion: "strong_harming", damage: 12},
	}
	for _, test := range tests {
		t.Run(test.potion, func(t *testing.T) {
			stack := ItemStack{ItemID: "minecraft:potion", Count: 1}
			if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:" + test.potion}); err != nil {
				t.Fatal(err)
			}
			outcome, ok := PotionOutcomeFor(stack)
			if !ok || outcome.Heal != test.heal || outcome.Damage != test.damage {
				t.Fatalf("outcome = %+v, ok=%v", outcome, ok)
			}
			if test.effect == "" {
				if len(outcome.Effects) != 0 {
					t.Fatalf("instant potion effects = %+v", outcome.Effects)
				}
				return
			}
			if len(outcome.Effects) != 1 || outcome.Effects[0].ID != test.effect ||
				outcome.Effects[0].Amplifier != test.amplifier || outcome.Effects[0].Duration != test.duration {
				t.Fatalf("effects = %+v", outcome.Effects)
			}
		})
	}
}

func TestPotionOutcomeIncludesCustomEffects(t *testing.T) {
	stack := ItemStack{ItemID: "minecraft:splash_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]any{
		"potion": "minecraft:water",
		"custom_effects": []map[string]any{
			{"id": "minecraft:haste", "amplifier": 2, "duration": 240},
			{"id": "minecraft:instant_health", "amplifier": 1, "duration": 1},
		},
	}); err != nil {
		t.Fatal(err)
	}
	outcome, ok := PotionOutcomeFor(stack)
	if !ok || outcome.Heal != 8 || len(outcome.Effects) != 1 {
		t.Fatalf("custom outcome = %+v, ok=%v", outcome, ok)
	}
	if effect := outcome.Effects[0]; effect.ID != "minecraft:haste" || effect.Amplifier != 2 || effect.Duration != 240 {
		t.Fatalf("custom effect = %+v", effect)
	}
	if _, ok := PotionOutcomeFor(ItemStack{ItemID: "minecraft:apple"}); ok {
		t.Fatal("non-potion stack produced a potion outcome")
	}
}

func TestPotionNameReadsCanonicalContents(t *testing.T) {
	stack := ItemStack{ItemID: "minecraft:lingering_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:long_poison"}); err != nil {
		t.Fatal(err)
	}
	if name, ok := PotionName(stack); !ok || name != "long_poison" {
		t.Fatalf("potion name = %q, ok=%v", name, ok)
	}
	if _, ok := PotionName(ItemStack{ItemID: "minecraft:apple", Count: 1}); ok {
		t.Fatal("non-potion stack reported a potion name")
	}
}

func TestPotionOutcomeForTurtleMasterAndTrialEffects(t *testing.T) {
	for _, name := range []string{"wind_charged", "weaving", "oozing", "infested"} {
		stack := ItemStack{ItemID: "minecraft:potion", Count: 1}
		if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:" + name}); err != nil {
			t.Fatal(err)
		}
		outcome, _ := PotionOutcomeFor(stack)
		if len(outcome.Effects) != 1 || outcome.Effects[0].ID != "minecraft:"+name || outcome.Effects[0].Duration != 3600 {
			t.Fatalf("%s outcome = %+v", name, outcome)
		}
	}
	stack := ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:strong_turtle_master"}); err != nil {
		t.Fatal(err)
	}
	outcome, _ := PotionOutcomeFor(stack)
	if len(outcome.Effects) != 2 || outcome.Effects[0].Amplifier != 3 || outcome.Effects[1].Amplifier != 5 {
		t.Fatalf("strong turtle master outcome = %+v", outcome)
	}
}
