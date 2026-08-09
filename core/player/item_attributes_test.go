package player

import "testing"

func TestLegacyWeaponDamageAndModernAttributes(t *testing.T) {
	if got := LegacyAttackDamage("minecraft:diamond_sword"); got != 7 {
		t.Fatalf("legacy diamond sword damage = %v, want 7", got)
	}
	if got := LegacyAttackDamage("minecraft:diamond_axe"); got != 6 {
		t.Fatalf("legacy diamond axe damage = %v, want 6", got)
	}
	damage, speed, ok := AttackAttributes("minecraft:netherite_pickaxe")
	if !ok || damage != 6 || speed != 1.2 {
		t.Fatalf("netherite pickaxe attributes = (%v,%v,%v), want (6,1.2,true)", damage, speed, ok)
	}
	if got := BlockUseDamage("minecraft:iron_sword"); got != 2 {
		t.Fatalf("sword block wear = %d, want 2", got)
	}
}

func TestNetheriteArmorSecondaryAttributes(t *testing.T) {
	p := New([16]byte{}, "armoured", ClientEditionJava)
	for slot, item := range []string{
		"minecraft:netherite_helmet", "minecraft:netherite_chestplate",
		"minecraft:netherite_leggings", "minecraft:netherite_boots",
	} {
		p.Inventory[5+slot] = ItemStack{ItemID: item, Count: 1}
	}
	if got := p.ArmorPoints(); got != 20 {
		t.Fatalf("armour = %d, want 20", got)
	}
	if got := p.ArmorToughness(); got != 12 {
		t.Fatalf("toughness = %v, want 12", got)
	}
	if got := p.KnockbackResistance(); got < 0.399 || got > 0.401 {
		t.Fatalf("knockback resistance = %v, want 0.4", got)
	}
}

func TestAllPumpkinArmorStats(t *testing.T) {
	want := map[string]armorItemStats{
		"minecraft:leather_helmet":         {maxDurability: 55, armor: 1},
		"minecraft:leather_chestplate":     {maxDurability: 80, armor: 3},
		"minecraft:leather_leggings":       {maxDurability: 75, armor: 2},
		"minecraft:leather_boots":          {maxDurability: 65, armor: 1},
		"minecraft:chainmail_helmet":       {maxDurability: 165, armor: 2},
		"minecraft:chainmail_chestplate":   {maxDurability: 240, armor: 5},
		"minecraft:chainmail_leggings":     {maxDurability: 225, armor: 4},
		"minecraft:chainmail_boots":        {maxDurability: 195, armor: 1},
		"minecraft:copper_helmet":          {maxDurability: 121, armor: 2},
		"minecraft:copper_chestplate":      {maxDurability: 176, armor: 4},
		"minecraft:copper_leggings":        {maxDurability: 165, armor: 3},
		"minecraft:copper_boots":           {maxDurability: 143, armor: 1},
		"minecraft:golden_helmet":          {maxDurability: 77, armor: 2},
		"minecraft:golden_chestplate":      {maxDurability: 112, armor: 5},
		"minecraft:golden_leggings":        {maxDurability: 105, armor: 3},
		"minecraft:golden_boots":           {maxDurability: 91, armor: 1},
		"minecraft:iron_helmet":            {maxDurability: 165, armor: 2},
		"minecraft:iron_chestplate":        {maxDurability: 240, armor: 6},
		"minecraft:iron_leggings":          {maxDurability: 225, armor: 5},
		"minecraft:iron_boots":             {maxDurability: 195, armor: 2},
		"minecraft:diamond_helmet":         {maxDurability: 363, armor: 3, toughness: 2},
		"minecraft:diamond_chestplate":     {maxDurability: 528, armor: 8, toughness: 2},
		"minecraft:diamond_leggings":       {maxDurability: 495, armor: 6, toughness: 2},
		"minecraft:diamond_boots":          {maxDurability: 429, armor: 3, toughness: 2},
		"minecraft:netherite_helmet":       {maxDurability: 407, armor: 3, toughness: 3, knockbackResistance: 0.1},
		"minecraft:netherite_chestplate":   {maxDurability: 592, armor: 8, toughness: 3, knockbackResistance: 0.1},
		"minecraft:netherite_leggings":     {maxDurability: 555, armor: 6, toughness: 3, knockbackResistance: 0.1},
		"minecraft:netherite_boots":        {maxDurability: 481, armor: 3, toughness: 3, knockbackResistance: 0.1},
		"minecraft:turtle_helmet":          {maxDurability: 275, armor: 2},
		"minecraft:wolf_armor":             {maxDurability: 64, armor: 11},
		"minecraft:leather_horse_armor":    {armor: 3},
		"minecraft:copper_horse_armor":     {armor: 4},
		"minecraft:iron_horse_armor":       {armor: 5},
		"minecraft:golden_horse_armor":     {armor: 7},
		"minecraft:diamond_horse_armor":    {armor: 11, toughness: 2},
		"minecraft:netherite_horse_armor":  {armor: 19, toughness: 3, knockbackResistance: 0.1},
		"minecraft:copper_nautilus_armor":  {armor: 4},
		"minecraft:iron_nautilus_armor":    {armor: 5},
		"minecraft:golden_nautilus_armor":  {armor: 7},
		"minecraft:diamond_nautilus_armor": {armor: 11, toughness: 2},
		"minecraft:netherite_nautilus_armor": {
			armor: 19, toughness: 3, knockbackResistance: 0.1,
		},
	}

	if len(armorItemStatsByID) != len(want) {
		t.Fatalf("armor stat table has %d entries, want %d from Pumpkin", len(armorItemStatsByID), len(want))
	}
	for itemID, expected := range want {
		if got := armorItemStatsByID[itemID]; got != expected {
			t.Errorf("%s stats = %+v, want %+v", itemID, got, expected)
		}
		if got := MaxDurability(itemID); got != expected.maxDurability {
			t.Errorf("%s durability = %d, want %d", itemID, got, expected.maxDurability)
		}
		if got := ArmorPoints(itemID); got != expected.armor {
			t.Errorf("%s armor = %d, want %d", itemID, got, expected.armor)
		}
		if got := ArmorToughness(itemID); got != expected.toughness {
			t.Errorf("%s toughness = %v, want %v", itemID, got, expected.toughness)
		}
		if got := ArmorKnockbackResistance(itemID); got != expected.knockbackResistance {
			t.Errorf("%s knockback resistance = %v, want %v", itemID, got, expected.knockbackResistance)
		}
		if got := MaxStackSize(itemID); got != 1 {
			t.Errorf("%s max stack = %d, want 1", itemID, got)
		}
	}
}
