package itemregistry

import "testing"

func TestRepresentativeDefinitions(t *testing.T) {
	tests := []struct {
		itemID     string
		durability int
		stack      int
		armor      int
		toughness  float32
		damage     float32
		speed      float32
		slot       string
		category   ToolCategory
	}{
		{"minecraft:leather_helmet", 55, 1, 1, 0, 0, 0, "head", ""},
		{"minecraft:iron_chestplate", 240, 1, 6, 0, 0, 0, "chest", ""},
		{"minecraft:diamond_leggings", 495, 1, 6, 2, 0, 0, "legs", ""},
		{"minecraft:netherite_boots", 481, 1, 3, 3, 0, 0, "feet", ""},
		{"minecraft:turtle_helmet", 275, 1, 2, 0, 0, 0, "head", ""},
		{"minecraft:wolf_armor", 64, 1, 11, 0, 0, 0, "body", ""},
		{"minecraft:wooden_sword", 59, 1, 0, 0, 4, 1.6, "", ToolSword},
		{"minecraft:iron_sword", 250, 1, 0, 0, 6, 1.6, "", ToolSword},
		{"minecraft:diamond_axe", 1561, 1, 0, 0, 9, 1, "", ToolAxe},
		{"minecraft:netherite_pickaxe", 2031, 1, 0, 0, 6, 1.2, "", ToolPickaxe},
		{"minecraft:trident", 250, 1, 0, 0, 9, 1.1, "", ToolTrident},
		{"minecraft:mace", 500, 1, 0, 0, 6, 0.6, "", ToolMace},
		{"minecraft:shears", 238, 1, 0, 0, 0, 0, "", ToolShears},
		{"minecraft:fishing_rod", 64, 1, 0, 0, 0, 0, "", ToolFishingRod},
		{"minecraft:brush", 64, 1, 0, 0, 0, 0, "", ToolBrush},
		{"minecraft:flint_and_steel", 64, 1, 0, 0, 0, 0, "", ToolFlintAndSteel},
		{"minecraft:bow", 384, 1, 0, 0, 0, 0, "", ""},
		{"minecraft:crossbow", 465, 1, 0, 0, 0, 0, "", ""},
		{"minecraft:shield", 336, 1, 0, 0, 0, 0, "", ""},
		{"minecraft:elytra", 432, 1, 0, 0, 0, 0, "chest", ""},
	}
	for _, test := range tests {
		t.Run(test.itemID, func(t *testing.T) {
			definition, ok := Lookup(test.itemID)
			if !ok {
				t.Fatal("definition not found")
			}
			if definition.MaxDurability != test.durability || definition.MaxStackSize != test.stack {
				t.Fatalf("durability/stack = %d/%d, want %d/%d", definition.MaxDurability, definition.MaxStackSize, test.durability, test.stack)
			}
			if test.slot != "" {
				if definition.Equipment == nil || definition.Equipment.Slot != test.slot || definition.Equipment.Armor != test.armor || definition.Equipment.Toughness != test.toughness {
					t.Fatalf("equipment = %+v, want slot=%s armor=%d toughness=%v", definition.Equipment, test.slot, test.armor, test.toughness)
				}
			}
			if test.damage != 0 {
				if definition.Combat == nil || definition.Combat.AttackDamage != test.damage || definition.Combat.AttackSpeed != test.speed {
					t.Fatalf("combat = %+v, want damage=%v speed=%v", definition.Combat, test.damage, test.speed)
				}
			}
			if test.category != "" && (definition.Tool == nil || definition.Tool.Category != test.category) {
				t.Fatalf("tool = %+v, want category %s", definition.Tool, test.category)
			}
		})
	}
}

func TestFoodStackSizesTagsAndExtensions(t *testing.T) {
	apple, ok := Lookup("minecraft:apple")
	if !ok || apple.Food == nil || apple.Food.Nutrition != 4 || apple.Food.SaturationModifier() != 0.3 || apple.MaxStackSize != 64 {
		t.Fatalf("apple = %+v", apple)
	}
	rabbitStew, ok := Lookup("minecraft:rabbit_stew")
	if !ok || rabbitStew.Food == nil || rabbitStew.Food.Nutrition != 10 || rabbitStew.Food.SaturationModifier() != 0.6 || rabbitStew.Consumable.Remainder != "minecraft:bowl" {
		t.Fatalf("rabbit stew = %+v", rabbitStew)
	}
	driedKelp, ok := Lookup("minecraft:dried_kelp")
	if !ok || driedKelp.Food == nil || driedKelp.Food.Nutrition != 1 || driedKelp.Consumable.UseDurationTicks != 16 {
		t.Fatalf("dried kelp = %+v", driedKelp)
	}
	if snowball, _ := Lookup("minecraft:snowball"); snowball.MaxStackSize != 16 {
		t.Fatalf("snowball stack = %d, want 16", snowball.MaxStackSize)
	}
	if !HasTag("minecraft:diamond_sword", "minecraft:enchantable/sharp_weapon") || HasTag("minecraft:bow", "minecraft:enchantable/sharp_weapon") {
		t.Fatal("sharp-weapon tag membership is wrong")
	}
	if !RepairsWith("minecraft:wooden_pickaxe", "minecraft:cherry_planks") || RepairsWith("minecraft:wooden_pickaxe", "minecraft:iron_ingot") {
		t.Fatal("wooden tool repair tag membership is wrong")
	}
	copper, ok := Lookup("minecraft:copper_helmet")
	if !ok || copper.MaxDurability != 121 || copper.Equipment == nil || copper.Equipment.Armor != 2 || copper.Equipment.Slot != "head" {
		t.Fatalf("copper compatibility definition = %+v", copper)
	}
	nautilus, ok := Lookup("minecraft:netherite_nautilus_armor")
	if !ok || nautilus.Equipment == nil || nautilus.Equipment.Armor != 19 || nautilus.Equipment.Toughness != 3 {
		t.Fatalf("nautilus compatibility definition = %+v", nautilus)
	}
}
