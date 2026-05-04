package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"GoCraft/core/player"
	"GoCraft/internal/gamedata"
)

// CookingRecipeDescription is the protocol-neutral form of one fixed Java
// 1.21.4 furnace-like recipe. Ingredient alternatives are already expanded
// from Java item tags by the authoritative recipe loader.
type CookingRecipeDescription struct {
	Name        string
	Station     string
	Ingredients []string
	Result      player.ItemStack
	CookingTime int
	Experience  float32
}

var (
	cookingRecipesOnce sync.Once
	cookingRecipes     map[string]map[string]CookingRecipeDescription
	furnaceFuelTicks   map[string]int
	furnaceFuelErr     error
)

func init() {
	var catalogue struct {
		Version string         `json:"_gocraft_version"`
		Fuels   map[string]int `json:"fuels"`
	}
	data, err := gamedata.FS.ReadFile("java/1.21.4/fuels.json")
	if err != nil {
		furnaceFuelErr = fmt.Errorf("reading embedded fuels: %w", err)
		return
	}
	if err := json.Unmarshal(data, &catalogue); err != nil {
		furnaceFuelErr = fmt.Errorf("decoding embedded fuels: %w", err)
		return
	}
	if catalogue.Version != RecipeCatalogVersion {
		furnaceFuelErr = fmt.Errorf("fuel catalogue version %q, want %s", catalogue.Version, RecipeCatalogVersion)
		return
	}
	furnaceFuelTicks = catalogue.Fuels
}

func buildCookingRecipeIndex() {
	cookingRecipes = make(map[string]map[string]CookingRecipeDescription)
	for _, recipe := range CraftingRecipeCatalog() {
		if recipe.Kind != "furnace" || len(recipe.Ingredients) != 1 || recipe.Result.IsEmpty() {
			continue
		}
		description := CookingRecipeDescription{
			Name: recipe.Name, Station: recipe.Station,
			Ingredients: append([]string(nil), recipe.Ingredients[0].Alternatives...),
			Result:      recipe.Result, CookingTime: int(recipe.CookingTime), Experience: recipe.Experience,
		}
		stationRecipes := cookingRecipes[recipe.Station]
		if stationRecipes == nil {
			stationRecipes = make(map[string]CookingRecipeDescription)
			cookingRecipes[recipe.Station] = stationRecipes
		}
		for _, ingredient := range description.Ingredients {
			stationRecipes[ingredient] = description
		}
	}
}

// FindCookingRecipe resolves a recipe for one furnace-family station and input.
func FindCookingRecipe(station, ingredient string) (CookingRecipeDescription, bool) {
	cookingRecipesOnce.Do(buildCookingRecipeIndex)
	station = normalizeFurnaceStation(station)
	recipe, ok := cookingRecipes[station][ingredient]
	return recipe, ok
}

// CookingRecipeCatalog returns detached cooking recipes, optionally limited to
// one station. An empty station returns every furnace-family recipe.
func CookingRecipeCatalog(station string) []CookingRecipeDescription {
	cookingRecipesOnce.Do(buildCookingRecipeIndex)
	station = normalizeFurnaceStation(station)
	result := make([]CookingRecipeDescription, 0)
	for currentStation, recipes := range cookingRecipes {
		if station != "" && currentStation != station {
			continue
		}
		seen := make(map[string]struct{})
		for _, recipe := range recipes {
			if _, exists := seen[recipe.Name]; exists {
				continue
			}
			seen[recipe.Name] = struct{}{}
			recipe.Ingredients = append([]string(nil), recipe.Ingredients...)
			result = append(result, recipe)
		}
	}
	return result
}

// FurnaceFuelDuration reports the vanilla Java 1.21.4 burn duration in ticks.
func FurnaceFuelDuration(itemID string) int {
	if furnaceFuelErr != nil {
		return 0
	}
	return furnaceFuelTicks[itemID]
}

// CanPlaceFurnaceFuelSlot reports whether an item is accepted by the fuel
// slot. Empty buckets are also valid for vanilla's wet-sponge interaction.
func CanPlaceFurnaceFuelSlot(itemID string) bool {
	return itemID == "minecraft:bucket" || FurnaceFuelDuration(itemID) > 0
}

// FurnaceFuelRemainder returns the container left after consuming a fuel item.
func FurnaceFuelRemainder(itemID string) player.ItemStack {
	if itemID == "minecraft:lava_bucket" {
		return player.ItemStack{ItemID: "minecraft:bucket", Count: 1}
	}
	return player.ItemStack{}
}

func normalizeFurnaceStation(station string) string {
	switch strings.TrimPrefix(station, "minecraft:") {
	case "furnace", "lit_furnace":
		return "minecraft:furnace"
	case "blast_furnace", "lit_blast_furnace":
		return "minecraft:blast_furnace"
	case "smoker", "lit_smoker":
		return "minecraft:smoker"
	case "campfire", "soul_campfire":
		return "minecraft:campfire"
	case "":
		return ""
	default:
		return station
	}
}

// IsFurnaceContainer reports whether a block uses the three-slot cooking UI.
func IsFurnaceContainer(blockID string) bool {
	station := normalizeFurnaceStation(blockID)
	return station == "minecraft:furnace" || station == "minecraft:blast_furnace" || station == "minecraft:smoker"
}
