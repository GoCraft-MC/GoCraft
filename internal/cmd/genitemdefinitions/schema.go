package main

type sourceDescription struct {
	Name         string `json:"name"`
	Version      string `json:"version,omitempty"`
	ArtifactSHA1 string `json:"artifact_sha1,omitempty"`
	InputSHA256  string `json:"input_sha256"`
	Description  string `json:"description"`
}

type catalogue struct {
	Schema                  int                   `json:"_schema"`
	MinecraftVersion        string                `json:"_minecraft_version"`
	Comment                 string                `json:"_comment"`
	Sources                 []sourceDescription   `json:"_sources"`
	CompatibilityExtensions []string              `json:"_compatibility_extensions"`
	Items                   map[string]definition `json:"items"`
	Tags                    map[string][]string   `json:"tags"`
}

type definition struct {
	MaxStackSize   int                   `json:"max_stack_size"`
	MaxDurability  int                   `json:"max_durability,omitempty"`
	Equipment      *equipmentProperties  `json:"equipment,omitempty"`
	Combat         *combatProperties     `json:"combat,omitempty"`
	Food           *foodProperties       `json:"food,omitempty"`
	Consumable     *consumableProperties `json:"consumable,omitempty"`
	Tool           *toolProperties       `json:"tool,omitempty"`
	Enchantability int                   `json:"enchantability,omitempty"`
	Repair         *repairProperties     `json:"repair,omitempty"`
	Rarity         string                `json:"rarity"`
	FireResistant  bool                  `json:"fire_resistant,omitempty"`
	FuelTicks      int                   `json:"fuel_ticks,omitempty"`
}

type equipmentProperties struct {
	Slot                string  `json:"slot"`
	Armor               int     `json:"armor,omitempty"`
	Toughness           float32 `json:"toughness,omitempty"`
	KnockbackResistance float32 `json:"knockback_resistance,omitempty"`
}

type combatProperties struct {
	AttackDamage float32 `json:"attack_damage"`
	AttackSpeed  float32 `json:"attack_speed"`
}

type foodProperties struct {
	Nutrition    int32   `json:"nutrition"`
	Saturation   float32 `json:"saturation"`
	AlwaysEdible bool    `json:"always_edible,omitempty"`
}

type consumableProperties struct {
	UseDurationTicks int    `json:"use_duration_ticks"`
	Animation        string `json:"animation,omitempty"`
	Remainder        string `json:"remainder,omitempty"`
}

type toolProperties struct {
	Category        string  `json:"category"`
	Tier            string  `json:"tier,omitempty"`
	MiningSpeed     float32 `json:"mining_speed,omitempty"`
	BlockDamageCost int     `json:"block_damage_cost"`
}

type repairProperties struct {
	Ingredient string `json:"ingredient"`
}
