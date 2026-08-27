package player

import "testing"

func TestJava1214EnchantmentCatalogue(t *testing.T) {
	if got := len(EnchantmentIDs()); got != 42 {
		t.Fatalf("enchantment count = %d, want 42", got)
	}
	sharpness, ok := EnchantmentByID("sharpness")
	if !ok || sharpness.MaxLevel != 5 || !sharpness.Supports("minecraft:diamond_axe") || sharpness.Supports("minecraft:bow") {
		t.Fatalf("invalid sharpness definition: %+v", sharpness)
	}
	windBurst, ok := EnchantmentByID("wind_burst")
	if !ok || windBurst.MaxLevel != 3 || !windBurst.Supports("minecraft:mace") {
		t.Fatalf("invalid wind burst definition: %+v", windBurst)
	}
	if _, ok := EnchantmentByID("lunge"); ok {
		t.Fatal("post-1.21.4 lunge enchantment was exposed")
	}
}

func TestEnchantmentCompatibilityIsSymmetric(t *testing.T) {
	infinity, _ := EnchantmentByID("infinity")
	mending, _ := EnchantmentByID("mending")
	for _, test := range []struct {
		candidate Enchantment
		existing  Enchantment
	}{
		{infinity, mending},
		{mending, infinity},
	} {
		stack := ItemStack{ItemID: "minecraft:bow", Count: 1}
		stack.Enchant(test.existing.ID, 1)
		if test.candidate.CompatibleWith(stack) {
			t.Fatalf("%s accepted with %s", test.candidate.ID, test.existing.ID)
		}
	}
}
