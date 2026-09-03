package handler

import "sync"

type smithingTransformKey struct {
	template string
	base     string
	addition string
}

var (
	smithingTransformsOnce sync.Once
	smithingTransforms     map[smithingTransformKey]string
	smithingIngredients    [3]map[string]struct{}
)

func smithingTransform(template, base, addition string) (string, bool) {
	smithingTransformsOnce.Do(buildSmithingTransforms)
	result, ok := smithingTransforms[smithingTransformKey{template, base, addition}]
	return result, ok
}

func smithingAccepts(slot int, itemID string) bool {
	smithingTransformsOnce.Do(buildSmithingTransforms)
	if slot < 0 || slot >= len(smithingIngredients) {
		return false
	}
	_, ok := smithingIngredients[slot][itemID]
	return ok
}

func buildSmithingTransforms() {
	smithingTransforms = make(map[smithingTransformKey]string)
	for slot := range smithingIngredients {
		smithingIngredients[slot] = make(map[string]struct{})
	}
	for _, recipe := range CraftingRecipeCatalog() {
		if recipe.Kind != "smithing" || len(recipe.Ingredients) != 3 {
			continue
		}
		for slot, ingredient := range recipe.Ingredients {
			for _, itemID := range ingredient.Alternatives {
				smithingIngredients[slot][itemID] = struct{}{}
			}
		}
		if recipe.Result.IsEmpty() {
			continue
		}
		for _, template := range recipe.Ingredients[0].Alternatives {
			for _, base := range recipe.Ingredients[1].Alternatives {
				for _, addition := range recipe.Ingredients[2].Alternatives {
					key := smithingTransformKey{template, base, addition}
					smithingTransforms[key] = recipe.Result.ItemID
				}
			}
		}
	}
}
