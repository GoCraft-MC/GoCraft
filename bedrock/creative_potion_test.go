package bedrock

import (
	"testing"

	"GoCraft/core/player"
)

func TestCreativePotionVariantsKeepCanonicalContents(t *testing.T) {
	l := &Listener{}
	l.initCreativeContent()
	for _, known := range l.creativeNames {
		if known.name != "minecraft:splash_potion" || known.components == "" {
			continue
		}
		stack := player.ItemStack{ItemID: known.name, Count: 1, Components: known.components}
		if name, _ := player.PotionName(stack); name == "strong_healing" {
			return
		}
	}
	t.Fatal("strong healing splash potion lost from Bedrock creative catalogue")
}
