// Package itemregistry provides protocol-neutral static item definitions.
// Vanilla definitions are generated from versioned game data; gameplay code
// should query this package instead of inferring properties from item names.
package itemregistry

// Definition is the canonical static description of an item. It deliberately
// contains data, not use-state machines or protocol IDs.
type Definition struct {
	ID             string
	MaxStackSize   int
	MaxDurability  int
	Equipment      *EquipmentProperties
	Combat         *CombatProperties
	Food           *FoodProperties
	Consumable     *ConsumableProperties
	Tool           *ToolProperties
	Enchantability int
	Repair         *RepairProperties
	Rarity         Rarity
	FireResistant  bool
	FuelTicks      int
	Tags           []string
}

// EquipmentProperties describes a static equipment slot and its attributes.
// Slots use vanilla names such as head, chest, legs, feet and body.
type EquipmentProperties struct {
	Slot                string  `json:"slot"`
	Armor               int     `json:"armor,omitempty"`
	Toughness           float32 `json:"toughness,omitempty"`
	KnockbackResistance float32 `json:"knockback_resistance,omitempty"`
}

// CombatProperties contains the effective base-player values displayed when
// an item is held, rather than the raw attribute modifier deltas.
type CombatProperties struct {
	AttackDamage float32 `json:"attack_damage"`
	AttackSpeed  float32 `json:"attack_speed"`
}

// FoodProperties contains vanilla nutrition and restored saturation points.
type FoodProperties struct {
	Nutrition    int32   `json:"nutrition"`
	Saturation   float32 `json:"saturation"`
	AlwaysEdible bool    `json:"always_edible,omitempty"`
}

// SaturationModifier returns the legacy modifier used by player hunger APIs.
func (f FoodProperties) SaturationModifier() float32 {
	if f.Nutrition <= 0 {
		return 0
	}
	return f.Saturation / (float32(f.Nutrition) * 2)
}

// ConsumableProperties contains static use timing and container information.
// Status-effect and interaction behavior intentionally remains gameplay code.
type ConsumableProperties struct {
	UseDurationTicks int    `json:"use_duration_ticks"`
	Animation        string `json:"animation,omitempty"`
	Remainder        string `json:"remainder,omitempty"`
}

// ToolProperties classifies tools without parsing their resource locations.
type ToolProperties struct {
	Category        ToolCategory `json:"category"`
	Tier            ToolTier     `json:"tier,omitempty"`
	MiningSpeed     float32      `json:"mining_speed,omitempty"`
	BlockDamageCost int          `json:"block_damage_cost"`
}

// RepairProperties identifies one item or #tag accepted by an anvil repair.
type RepairProperties struct {
	Ingredient string `json:"ingredient"`
}
