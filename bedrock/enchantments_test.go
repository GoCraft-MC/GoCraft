package bedrock

import (
	"testing"

	"GoCraft/core/player"
)

func TestBedrockEnchantmentNBTIDs(t *testing.T) {
	stack := player.ItemStack{
		ItemID: "minecraft:mace", Count: 1,
		Enchantments: "minecraft:density=5;minecraft:wind_burst=3",
	}
	values := bedrockEnchantments(stack)
	if len(values) != 2 {
		t.Fatalf("enchantments = %#v", values)
	}
	if values[0]["id"] != int16(39) || values[0]["lvl"] != int16(5) {
		t.Fatalf("density NBT = %#v", values[0])
	}
	if values[1]["id"] != int16(38) || values[1]["lvl"] != int16(3) {
		t.Fatalf("wind burst NBT = %#v", values[1])
	}
}
