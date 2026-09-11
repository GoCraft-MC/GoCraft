package player

import "testing"

func TestConsumableCleansing(t *testing.T) {
	p := New([16]byte{}, "drinker", ClientEditionJava)
	p.AddStatusEffect(StatusEffect{ID: "poison", Duration: 100})
	p.AddStatusEffect(StatusEffect{ID: "speed", Duration: 100})
	removed := p.ApplyConsumableCleansing("minecraft:honey_bottle")
	if len(removed) != 1 || removed[0].ID != "minecraft:poison" {
		t.Fatalf("honey removed %+v", removed)
	}
	if _, ok := p.StatusEffect("speed"); !ok {
		t.Fatal("honey removed a non-poison effect")
	}
	p.AddStatusEffect(StatusEffect{ID: "poison", Duration: 100})
	if removed = p.ApplyConsumableCleansing("minecraft:milk_bucket"); len(removed) != 2 {
		t.Fatalf("milk removed %+v", removed)
	}
	if effects := p.StatusEffectsSnapshot(); len(effects) != 0 {
		t.Fatalf("effects remain after milk: %+v", effects)
	}
}
