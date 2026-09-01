package main

const (
	minecraftVersion    = "1.21.4"
	minecraftServerSHA1 = "4707d00eb834b446575d89a61a11b5d548d8c001"
)

// GoCraft already exposes these newer items through its Bedrock compatibility
// data. They form an explicit extension to the strict Java 1.21.4 base set.
var compatibilityExtensionIDs = []string{
	"minecraft:copper_helmet",
	"minecraft:copper_chestplate",
	"minecraft:copper_leggings",
	"minecraft:copper_boots",
	"minecraft:copper_horse_armor",
	"minecraft:netherite_horse_armor",
	"minecraft:copper_nautilus_armor",
	"minecraft:iron_nautilus_armor",
	"minecraft:golden_nautilus_armor",
	"minecraft:diamond_nautilus_armor",
	"minecraft:netherite_nautilus_armor",
}
