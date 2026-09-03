package itemregistry

import (
	"encoding/json"
	"fmt"

	"GoCraft/internal/gamedata"
)

type dataDefinition struct {
	MaxStackSize   int                   `json:"max_stack_size"`
	MaxDurability  int                   `json:"max_durability,omitempty"`
	Equipment      *EquipmentProperties  `json:"equipment,omitempty"`
	Combat         *CombatProperties     `json:"combat,omitempty"`
	Food           *FoodProperties       `json:"food,omitempty"`
	Consumable     *ConsumableProperties `json:"consumable,omitempty"`
	Tool           *ToolProperties       `json:"tool,omitempty"`
	Enchantability int                   `json:"enchantability,omitempty"`
	Repair         *RepairProperties     `json:"repair,omitempty"`
	Rarity         Rarity                `json:"rarity"`
	FireResistant  bool                  `json:"fire_resistant,omitempty"`
	FuelTicks      int                   `json:"fuel_ticks,omitempty"`
}

type dataCatalogue struct {
	Schema           int                       `json:"_schema"`
	MinecraftVersion string                    `json:"_minecraft_version"`
	Items            map[string]dataDefinition `json:"items"`
	Tags             map[string][]string       `json:"tags"`
}

func mustLoadVanilla() *Registry {
	data, err := gamedata.FS.ReadFile("vanilla/1.21.4/item_definitions.json")
	if err != nil {
		panic(fmt.Sprintf("itemregistry: reading item definitions: %v", err))
	}
	var catalogue dataCatalogue
	if err := json.Unmarshal(data, &catalogue); err != nil {
		panic(fmt.Sprintf("itemregistry: parsing item definitions: %v", err))
	}
	if catalogue.Schema != 1 || catalogue.MinecraftVersion != vanillaVersion {
		panic(fmt.Sprintf("itemregistry: incompatible schema/version %d/%q", catalogue.Schema, catalogue.MinecraftVersion))
	}
	reverseTags := make(map[string][]string, len(catalogue.Items))
	for tag, members := range catalogue.Tags {
		for _, itemID := range members {
			if _, ok := catalogue.Items[itemID]; !ok {
				panic(fmt.Sprintf("itemregistry: tag %s references unknown item %s", tag, itemID))
			}
			reverseTags[itemID] = append(reverseTags[itemID], tag)
		}
	}
	definitions := make([]Definition, 0, len(catalogue.Items))
	for itemID, data := range catalogue.Items {
		definitions = append(definitions, Definition{
			ID: itemID, MaxStackSize: data.MaxStackSize, MaxDurability: data.MaxDurability,
			Equipment: data.Equipment, Combat: data.Combat, Food: data.Food,
			Consumable: data.Consumable, Tool: data.Tool, Enchantability: data.Enchantability,
			Repair: data.Repair, Rarity: data.Rarity, FireResistant: data.FireResistant,
			FuelTicks: data.FuelTicks, Tags: reverseTags[itemID],
		})
	}
	registry, err := NewRegistry(definitions)
	if err != nil {
		panic(err)
	}
	if len(registry.definitions) < 1385 {
		panic(fmt.Sprintf("itemregistry: only %d definitions loaded", len(registry.definitions)))
	}
	return registry
}
