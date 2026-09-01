package itemregistry

func cloneDefinition(source Definition) Definition {
	result := source
	result.Tags = append([]string(nil), source.Tags...)
	if source.Equipment != nil {
		value := *source.Equipment
		result.Equipment = &value
	}
	if source.Combat != nil {
		value := *source.Combat
		result.Combat = &value
	}
	if source.Food != nil {
		value := *source.Food
		result.Food = &value
	}
	if source.Consumable != nil {
		value := *source.Consumable
		result.Consumable = &value
	}
	if source.Tool != nil {
		value := *source.Tool
		result.Tool = &value
	}
	if source.Repair != nil {
		value := *source.Repair
		result.Repair = &value
	}
	return result
}
