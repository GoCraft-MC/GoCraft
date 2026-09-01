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
)

func smithingTransform(template, base, addition string) (string, bool) {
	smithingTransformsOnce.Do(buildSmithingTransforms)
	result, ok := smithingTransforms[smithingTransformKey{template, base, addition}]
	return result, ok
}

func buildSmithingTransforms() {
	smithingTransforms = make(map[smithingTransformKey]string)
	for _, recipe := range CraftingRecipeCatalog() {
		if recipe.Kind != "smithing" || len(recipe.Ingredients) != 3 || recipe.Result.IsEmpty() {
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
