package bedrock

import (
	"testing"

	"GoCraft/core/player"
)

func TestBedrockStewVariantRoundTrip(t *testing.T) {
	for variant := int16(0); int(variant) < len(bedrockStewEffects); variant++ {
		stack := player.ItemStack{ItemID: "minecraft:suspicious_stew", Count: 1}
		if !setBedrockStewContents(&stack, variant) {
			t.Fatalf("could not decode stew variant %d", variant)
		}
		effects := player.SuspiciousStewEffects(stack)
		if len(effects) != 1 || effects[0].ID != bedrockStewEffects[variant].ID ||
			effects[0].Duration != bedrockStewEffects[variant].Duration {
			t.Fatalf("stew variant %d decoded as %v", variant, effects)
		}
		// Several flowers share an effect (e.g. two saturation variants), so the
		// canonical component only stores the effect and re-encoding may pick the
		// first matching variant. Require the encoded variant's effect to match.
		id, ok := bedrockStewVariant(stack)
		if !ok || bedrockStewEffects[id] != bedrockStewEffects[variant] {
			t.Fatalf("stew variant %d encoded as %d (effect %v), ok=%v",
				variant, id, bedrockStewEffects[id], ok)
		}
	}
}

func TestBedrockStewVariantRejectsUnknownValues(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:suspicious_stew", Count: 1}
	if setBedrockStewContents(&stack, int16(len(bedrockStewEffects))) {
		t.Fatal("out-of-range Bedrock stew variant was accepted")
	}
	if err := stack.SetComponent("suspicious_stew_effects", []player.StatusEffect{
		{ID: "minecraft:unknown", Duration: 999},
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := bedrockStewVariant(stack); ok {
		t.Fatal("unknown canonical stew effect was encoded")
	}
}

func TestBedrockStewVariantEmptyStewMapsToBase(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:suspicious_stew", Count: 1}
	if id, ok := bedrockStewVariant(stack); !ok || id != 0 {
		t.Fatalf("empty stew encoded as %d, ok=%v", id, ok)
	}
}
