package bedrock

import (
	"testing"

	"GoCraft/core/player"
)

func TestBedrockPotionVariantRoundTrip(t *testing.T) {
	for _, test := range []struct {
		id   int16
		name string
	}{{0, "water"}, {22, "strong_healing"}, {42, "strong_slowness"}, {46, "infested"}} {
		stack := player.ItemStack{ItemID: "minecraft:splash_potion", Count: 1}
		if !setBedrockPotionContents(&stack, test.id) {
			t.Fatalf("could not decode potion ID %d", test.id)
		}
		if name, _ := player.PotionName(stack); name != test.name {
			t.Fatalf("potion ID %d decoded as %q", test.id, name)
		}
		if id, ok := bedrockPotionID(stack); !ok || id != test.id {
			t.Fatalf("potion %q encoded as %d, ok=%v", test.name, id, ok)
		}
	}
}

func TestBedrockPotionVariantRejectsUnknownValues(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:potion", Count: 1}
	if setBedrockPotionContents(&stack, 47) {
		t.Fatal("unknown Bedrock potion ID was accepted")
	}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:unknown"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := bedrockPotionID(stack); ok {
		t.Fatal("unknown canonical potion was encoded")
	}
}
