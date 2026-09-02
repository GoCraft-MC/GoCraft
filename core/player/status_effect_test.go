package player

import "testing"

func TestStatusEffectUpgradeAndRemoval(t *testing.T) {
	p := New([16]byte{1}, "effects", ClientEditionJava)
	first := StatusEffect{ID: "absorption", Duration: 100, ShowParticles: true, ShowIcon: true}
	if _, changed := p.AddStatusEffect(first); !changed || p.Absorption != 4 {
		t.Fatalf("first absorption = %.1f, changed=%v", p.Absorption, changed)
	}
	if _, changed := p.AddStatusEffect(StatusEffect{ID: "absorption", Duration: 50}); changed {
		t.Fatal("shorter equal-strength effect replaced active effect")
	}
	if effect, changed := p.AddStatusEffect(StatusEffect{ID: "absorption", Amplifier: 1, Duration: 40}); !changed || effect.Amplifier != 1 || p.Absorption != 8 {
		t.Fatalf("upgrade = %#v, absorption %.1f, changed=%v", effect, p.Absorption, changed)
	}
	if _, removed := p.RemoveStatusEffect("absorption"); !removed || p.Absorption != 0 {
		t.Fatalf("remove left absorption %.1f, removed=%v", p.Absorption, removed)
	}
}

func TestStatusEffectsTickActionsAndExpiry(t *testing.T) {
	p := New([16]byte{2}, "tick", ClientEditionBedrock)
	p.Health = 10
	for _, effect := range []StatusEffect{
		{ID: "regeneration", Duration: 50},
		{ID: "poison", Duration: 25},
		{ID: "hunger", Amplifier: 1, Duration: 1},
	} {
		if _, ok := p.AddStatusEffect(effect); !ok {
			t.Fatalf("could not add %#v", effect)
		}
	}
	result := p.TickStatusEffects()
	if result.Heal != 1 || result.Damage != 1 || result.CanKill || result.Exhaustion != 0.01 {
		t.Fatalf("tick result = %#v", result)
	}
	if len(result.Expired) != 1 || result.Expired[0].ID != "minecraft:hunger" {
		t.Fatalf("expired effects = %#v", result.Expired)
	}
}

func TestStatusEffectMovementMultiplier(t *testing.T) {
	p := New([16]byte{3}, "runner", ClientEditionJava)
	p.AddStatusEffect(StatusEffect{ID: "speed", Amplifier: 1, Duration: 20})
	p.AddStatusEffect(StatusEffect{ID: "slowness", Duration: 20})
	if got := p.MovementSpeedMultiplier(); got < 1.189 || got > 1.191 {
		t.Fatalf("movement multiplier = %f, want 1.19", got)
	}
	removed := p.ClearStatusEffects()
	if len(removed) != 2 || len(p.StatusEffectsSnapshot()) != 0 {
		t.Fatalf("clear returned %#v, remaining %#v", removed, p.StatusEffectsSnapshot())
	}
}
