package player

import "testing"

func TestSuspiciousStewEffectsReadPerStackPayload(t *testing.T) {
	stack := ItemStack{ItemID: "minecraft:suspicious_stew", Count: 1}
	if err := stack.SetComponent("suspicious_stew_effects", []StatusEffect{
		{ID: "night_vision", Duration: 100},
		{ID: "", Duration: 20},
	}); err != nil {
		t.Fatal(err)
	}
	effects := SuspiciousStewEffects(stack)
	if len(effects) != 1 || effects[0].ID != "minecraft:night_vision" || effects[0].Duration != 100 {
		t.Fatalf("stew effects = %+v", effects)
	}
	if !effects[0].ShowParticles || !effects[0].ShowIcon {
		t.Fatalf("stew effect visibility = %+v", effects[0])
	}
	if effects := SuspiciousStewEffects(ItemStack{ItemID: "minecraft:mushroom_stew", Count: 1}); effects != nil {
		t.Fatalf("ordinary stew effects = %+v", effects)
	}
}
