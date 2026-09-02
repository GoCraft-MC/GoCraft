package bedrock

import (
	"testing"

	bedrockworld "GoCraft/bedrock/world"
	"GoCraft/core/player"
)

func TestBedrockExtensionComponentsArePreservedInNBT(t *testing.T) {
	stack := player.ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:healing"}); err != nil {
		t.Fatal(err)
	}
	instance := (&Listener{encoder: bedrockworld.NewEncoder()}).itemInstance(stack, 7)
	if got := instance.Stack.NBTData[goCraftComponentsNBTKey]; got != stack.Components {
		t.Fatalf("Bedrock component NBT = %#v, want %q", got, stack.Components)
	}
}
