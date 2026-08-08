package bedrock

import (
	"fmt"
	"strings"

	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"

	"GoCraft/core/player"
	"GoCraft/java/handler"
)

const bedrockRecipeCompatibilityVersion = "1.21.4"

// bedrockCraftingCatalogue keeps the Java catalogue authoritative and expands
// Java ingredient alternatives into concrete Bedrock recipe variants. Every
// recipe carries an inline AlwaysUnlocked requirement. Keeping every descriptor
// in Pumpkin's concrete "name" form is important: Some Bedrock clients crash
// when opening the recipe UI if Java item-tag descriptors are included.
func bedrockCraftingCatalogue() *packet.CraftingData {
	if handler.RecipeCatalogVersion != bedrockRecipeCompatibilityVersion {
		panic(fmt.Sprintf("bedrock recipes require Java %s catalogue, got %s",
			bedrockRecipeCompatibilityVersion, handler.RecipeCatalogVersion))
	}
	dfworld.DefaultBlockRegistry.Finalize()

	data := &packet.CraftingData{ClearRecipes: false}
	unlock := protocol.Option(protocol.RecipeUnlockRequirement{
		Context: protocol.RecipeUnlockContextAlwaysUnlocked,
	})
	var networkID uint32 = 1
	for _, recipe := range handler.CraftingRecipeCatalog() {
		if recipe.Kind != "shaped" && recipe.Kind != "shapeless" && recipe.Kind != "furnace" {
			continue
		}
		variants := bedrockJavaIngredientVariants(recipe)
		if len(variants) == 0 {
			continue
		}
		output, ok := bedrockJavaOutput(recipe.Result)
		if !ok {
			continue
		}

		for _, variant := range variants {
			switch recipe.Kind {
			case "shaped":
				if recipe.Width <= 0 || recipe.Height <= 0 || int(recipe.Width*recipe.Height) != len(variant.inputs) {
					continue
				}
				data.ShapedRecipes = append(data.ShapedRecipes, protocol.ShapedRecipe{
					RecipeID:          variant.recipeID,
					Width:             recipe.Width,
					Height:            recipe.Height,
					Input:             variant.inputs,
					Output:            []protocol.ItemStack{output},
					Block:             "crafting_table",
					Priority:          1,
					AssumeSymmetry:    true,
					UnlockRequirement: unlock,
					RecipeNetworkID:   networkID,
				})
			case "shapeless", "furnace":
				block := "crafting_table"
				if recipe.Kind == "furnace" {
					block = strings.TrimPrefix(recipe.Station, "minecraft:")
				}
				data.ShapelessRecipes = append(data.ShapelessRecipes, protocol.ShapelessRecipe{
					RecipeID:          variant.recipeID,
					Input:             variant.inputs,
					Output:            []protocol.ItemStack{output},
					Block:             block,
					Priority:          1,
					UnlockRequirement: unlock,
					RecipeNetworkID:   networkID,
				})
			}
			networkID++
		}
	}
	return data
}

const maxRecipesPerCraftingDataPacket = 1024

func bedrockCraftingData() []*packet.CraftingData {
	catalogue := bedrockCraftingCatalogue()
	packetCount := max(
		(len(catalogue.ShapedRecipes)+maxRecipesPerCraftingDataPacket-1)/maxRecipesPerCraftingDataPacket,
		(len(catalogue.ShapelessRecipes)+maxRecipesPerCraftingDataPacket-1)/maxRecipesPerCraftingDataPacket,
	)
	if packetCount == 0 {
		return []*packet.CraftingData{{ClearRecipes: false}}
	}
	packets := make([]*packet.CraftingData, packetCount)
	for index := range packets {
		packets[index] = &packet.CraftingData{ClearRecipes: false}
		shapedStart := index * maxRecipesPerCraftingDataPacket
		if shapedStart < len(catalogue.ShapedRecipes) {
			shapedEnd := min(shapedStart+maxRecipesPerCraftingDataPacket, len(catalogue.ShapedRecipes))
			packets[index].ShapedRecipes = catalogue.ShapedRecipes[shapedStart:shapedEnd]
		}
		shapelessStart := index * maxRecipesPerCraftingDataPacket
		if shapelessStart < len(catalogue.ShapelessRecipes) {
			shapelessEnd := min(shapelessStart+maxRecipesPerCraftingDataPacket, len(catalogue.ShapelessRecipes))
			packets[index].ShapelessRecipes = catalogue.ShapelessRecipes[shapelessStart:shapelessEnd]
		}
	}
	return packets
}

type bedrockIngredientVariant struct {
	recipeID string
	inputs   []protocol.ItemDescriptorCount
}

func bedrockJavaIngredientVariants(recipe handler.RecipeDescription) []bedrockIngredientVariant {
	// Bedrock's tag descriptor crashes some clients, so expand each distinct
	// Java alternative set into concrete variants. Repeated uses of the same tag
	// share one choice: Two #minecraft:planks cells therefore produce one recipe
	// per plank type instead of an unsafe tag descriptor or an exponential list
	// of every mixed-plank permutation.
	type alternativeGroup struct {
		alternatives []string
		selected     int
	}
	groups := make([]alternativeGroup, 0)
	ingredientGroups := make([]int, len(recipe.Ingredients))
	for index := range ingredientGroups {
		ingredientGroups[index] = -1
	}
	groupByAlternatives := make(map[string]int)
	for index, ingredient := range recipe.Ingredients {
		if len(ingredient.Alternatives) <= 1 {
			continue
		}
		key := strings.Join(ingredient.Alternatives, "\x00")
		group, found := groupByAlternatives[key]
		if !found {
			group = len(groups)
			groupByAlternatives[key] = group
			groups = append(groups, alternativeGroup{alternatives: ingredient.Alternatives})
		}
		ingredientGroups[index] = group
	}

	variants := make([]bedrockIngredientVariant, 0, 1)
	variantNumber := 0
	for {
		inputs := make([]protocol.ItemDescriptorCount, 0, len(recipe.Ingredients))
		valid := true
		for index, ingredient := range recipe.Ingredients {
			if len(ingredient.Alternatives) == 0 {
				inputs = append(inputs, protocol.ItemDescriptorCount{Descriptor: &protocol.InvalidItemDescriptor{}})
				continue
			}
			alternative := ingredient.Alternatives[0]
			if group := ingredientGroups[index]; group >= 0 {
				alternative = groups[group].alternatives[groups[group].selected]
			}
			name, metadata, ok := bedrockItemIdentity(alternative)
			if !ok {
				valid = false
				break
			}
			inputs = append(inputs, protocol.ItemDescriptorCount{
				Descriptor: &protocol.DefaultItemDescriptor{Name: name, MetadataValue: int32(metadata)},
				Count:      1,
			})
		}
		if valid {
			recipeID := recipe.Name
			if variantNumber > 0 {
				path := strings.TrimPrefix(recipe.Name, "minecraft:")
				recipeID = fmt.Sprintf("gocraft:java_1_21_4/%s/%d", path, variantNumber)
			}
			variants = append(variants, bedrockIngredientVariant{recipeID: recipeID, inputs: inputs})
		}

		variantNumber++
		group := len(groups) - 1
		for ; group >= 0; group-- {
			groups[group].selected++
			if groups[group].selected < len(groups[group].alternatives) {
				break
			}
			groups[group].selected = 0
		}
		if group < 0 {
			break
		}
	}
	return variants
}

func bedrockJavaIngredients(ingredients []handler.RecipeIngredientDescription) ([]protocol.ItemDescriptorCount, bool) {
	result := make([]protocol.ItemDescriptorCount, 0, len(ingredients))
	for _, ingredient := range ingredients {
		if len(ingredient.Alternatives) == 0 {
			result = append(result, protocol.ItemDescriptorCount{
				Descriptor: &protocol.InvalidItemDescriptor{},
			})
			continue
		}
		// Pumpkin deliberately chooses the first concrete member of a Java tag
		// or alternative list. The Java catalogue orders these deterministically.
		name, metadata, ok := bedrockItemIdentity(ingredient.Alternatives[0])
		if !ok {
			return nil, false
		}
		result = append(result, protocol.ItemDescriptorCount{
			Descriptor: &protocol.DefaultItemDescriptor{Name: name, MetadataValue: int32(metadata)},
			Count:      1,
		})
	}
	return result, true
}

func bedrockItemIdentity(javaName string) (string, int16, bool) {
	if mapping, ok := javaToBedrockItemMappings[javaName]; ok {
		return mapping.name, int16(mapping.metadata), true
	}
	// The Bedrock runtime registry contains more vanilla entries than
	// Dragonfly implements as concrete Go item types. Resolve by encoded name,
	// exactly as normal GoCraft inventory synchronisation does.
	_, metadata, ok := dfworld.ItemRuntimeID(namedItem{name: javaName})
	if !ok {
		return "", 0, false
	}
	return javaName, metadata, true
}

func bedrockJavaOutput(stack player.ItemStack) (protocol.ItemStack, bool) {
	if stack.IsEmpty() || stack.Count > int(^uint16(0)) {
		return protocol.ItemStack{}, false
	}
	mapping, mapped := javaToBedrockItemMappings[stack.ItemID]
	var runtimeID, blockRuntimeID int32
	var metadata uint32
	if mapped {
		runtimeID = mapping.runtimeID
		metadata = mapping.metadata
		blockRuntimeID = mapping.blockRuntimeID
	} else {
		var meta int16
		var ok bool
		runtimeID, meta, ok = dfworld.ItemRuntimeID(namedItem{name: stack.ItemID, meta: int16(stack.Damage)})
		if !ok {
			return protocol.ItemStack{}, false
		}
		metadata = uint32(uint16(meta))
		if item, found := dfworld.ItemByName(stack.ItemID, int16(stack.Damage)); found {
			if block, isBlock := item.(dfworld.Block); isBlock {
				blockRuntimeID = creativeBlockNetworkID(block)
			}
		}
	}
	return protocol.ItemStack{
		ItemType: protocol.ItemType{
			NetworkID:     runtimeID,
			MetadataValue: metadata,
		},
		Count:          uint16(stack.Count),
		BlockRuntimeID: blockRuntimeID,
	}, true
}
