package player

import "testing"

func TestEnchantReplacesOnlyWithHigherLevel(t *testing.T) {
	stack := ItemStack{ItemID: "minecraft:diamond_sword", Count: 1}
	if !stack.Enchant("unbreaking", 2) || !stack.Enchant("sharpness", 3) {
		t.Fatal("valid enchantment was rejected")
	}
	if stack.Enchant("sharpness", 2) {
		t.Fatal("lower enchantment level replaced the existing level")
	}
	if got := stack.EnchantmentLevel("minecraft:sharpness"); got != 3 {
		t.Fatalf("sharpness level = %d", got)
	}
	if stack.Enchantments != "minecraft:sharpness=3;minecraft:unbreaking=2" {
		t.Fatalf("component is not sorted: %q", stack.Enchantments)
	}
}
