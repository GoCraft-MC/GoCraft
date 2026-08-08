package handler

import "testing"

func TestJava1214CookingCataloguesAndFuelTable(t *testing.T) {
	for station, want := range map[string]int{
		"minecraft:furnace":       71,
		"minecraft:blast_furnace": 24,
		"minecraft:smoker":        9,
	} {
		if got := len(CookingRecipeCatalog(station)); got != want {
			t.Fatalf("%s recipes = %d, want %d Java 1.21.4 recipes", station, got, want)
		}
	}
	recipe, ok := FindCookingRecipe("minecraft:furnace", "minecraft:raw_iron")
	if !ok || recipe.Result.ItemID != "minecraft:iron_ingot" || recipe.CookingTime != 200 {
		t.Fatalf("raw iron furnace recipe = %+v, found=%v", recipe, ok)
	}
	if _, ok := FindCookingRecipe("minecraft:smoker", "minecraft:raw_iron"); ok {
		t.Fatal("smoker incorrectly accepts the furnace raw-iron recipe")
	}
	for item, want := range map[string]int{
		"minecraft:coal":             1600,
		"minecraft:coal_block":       16000,
		"minecraft:lava_bucket":      20000,
		"minecraft:oak_planks":       300,
		"minecraft:oak_slab":         150,
		"minecraft:stick":            100,
		"minecraft:dried_kelp_block": 4001,
	} {
		if got := FurnaceFuelDuration(item); got != want {
			t.Errorf("fuel duration %s = %d, want %d", item, got, want)
		}
	}
	if got := FurnaceFuelDuration("minecraft:sulfur"); got != 0 {
		t.Fatalf("post-1.21.4 Pumpkin item leaked into fuel catalogue: %d", got)
	}
}
