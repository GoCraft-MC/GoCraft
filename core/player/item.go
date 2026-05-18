package player

import "strings"

// InventorySize is the total number of slots in a Java Edition player inventory.
//
// Slot layout:
//
//	0      crafting output
//	1–4    2×2 crafting grid
//	5–8    armour (helmet, chestplate, leggings, boots)
//	9–35   main inventory (3 rows × 9)
//	36–44  hotbar (9 slots; index = HotbarStart + heldSlot)
//	45     off-hand
const InventorySize = 46

// HotbarStart is the slot index of the first hotbar slot.
const HotbarStart = 36

// OffhandSlot is separate from the 36-slot pickup/storage inventory. It must
// never be considered an empty destination by GiveItem: Bedrock only permits a
// limited set of items there, and server-forcing arbitrary drops into it makes
// them appear stuck or desynchronised in the client UI.
const OffhandSlot = 45

// ItemStack is a quantity of one item type occupying a single inventory slot.
// A zero-value ItemStack (or one with Count ≤ 0) represents an empty slot.
//
// ItemID uses the Minecraft resource-location format ("namespace:name"),
// e.g. "minecraft:stone".  Edition-specific numeric item IDs are resolved at
// the Java adapter boundary (java/world/items.go) and are not stored here.
type ItemStack struct {
	// ItemID is the canonical resource location of the item, e.g. "minecraft:stone".
	// Empty string means the slot is empty.
	ItemID string
	// Count is the number of items in the stack.
	Count int
	// Damage is durability already consumed. New items start at zero and break
	// when Damage reaches MaxDurability(ItemID).
	Damage int
}

type armorItemStats struct {
	maxDurability       int
	armor               int
	toughness           float32
	knockbackResistance float32
}

// armorItemStatsByID mirrors Pumpkin's generated MaxDamage and
// AttributeModifiers components. Keeping player and mount armour together
// makes stack limits and future entity-equipment handling use the same source
// of truth as player combat.
var armorItemStatsByID = map[string]armorItemStats{
	"minecraft:leather_helmet":     {maxDurability: 55, armor: 1},
	"minecraft:leather_chestplate": {maxDurability: 80, armor: 3},
	"minecraft:leather_leggings":   {maxDurability: 75, armor: 2},
	"minecraft:leather_boots":      {maxDurability: 65, armor: 1},

	"minecraft:chainmail_helmet":     {maxDurability: 165, armor: 2},
	"minecraft:chainmail_chestplate": {maxDurability: 240, armor: 5},
	"minecraft:chainmail_leggings":   {maxDurability: 225, armor: 4},
	"minecraft:chainmail_boots":      {maxDurability: 195, armor: 1},

	"minecraft:copper_helmet":     {maxDurability: 121, armor: 2},
	"minecraft:copper_chestplate": {maxDurability: 176, armor: 4},
	"minecraft:copper_leggings":   {maxDurability: 165, armor: 3},
	"minecraft:copper_boots":      {maxDurability: 143, armor: 1},

	"minecraft:golden_helmet":     {maxDurability: 77, armor: 2},
	"minecraft:golden_chestplate": {maxDurability: 112, armor: 5},
	"minecraft:golden_leggings":   {maxDurability: 105, armor: 3},
	"minecraft:golden_boots":      {maxDurability: 91, armor: 1},

	"minecraft:iron_helmet":     {maxDurability: 165, armor: 2},
	"minecraft:iron_chestplate": {maxDurability: 240, armor: 6},
	"minecraft:iron_leggings":   {maxDurability: 225, armor: 5},
	"minecraft:iron_boots":      {maxDurability: 195, armor: 2},

	"minecraft:diamond_helmet":     {maxDurability: 363, armor: 3, toughness: 2},
	"minecraft:diamond_chestplate": {maxDurability: 528, armor: 8, toughness: 2},
	"minecraft:diamond_leggings":   {maxDurability: 495, armor: 6, toughness: 2},
	"minecraft:diamond_boots":      {maxDurability: 429, armor: 3, toughness: 2},

	"minecraft:netherite_helmet":     {maxDurability: 407, armor: 3, toughness: 3, knockbackResistance: 0.1},
	"minecraft:netherite_chestplate": {maxDurability: 592, armor: 8, toughness: 3, knockbackResistance: 0.1},
	"minecraft:netherite_leggings":   {maxDurability: 555, armor: 6, toughness: 3, knockbackResistance: 0.1},
	"minecraft:netherite_boots":      {maxDurability: 481, armor: 3, toughness: 3, knockbackResistance: 0.1},

	"minecraft:turtle_helmet": {maxDurability: 275, armor: 2},
	"minecraft:wolf_armor":    {maxDurability: 64, armor: 11},

	"minecraft:leather_horse_armor":   {armor: 3},
	"minecraft:copper_horse_armor":    {armor: 4},
	"minecraft:iron_horse_armor":      {armor: 5},
	"minecraft:golden_horse_armor":    {armor: 7},
	"minecraft:diamond_horse_armor":   {armor: 11, toughness: 2},
	"minecraft:netherite_horse_armor": {armor: 19, toughness: 3, knockbackResistance: 0.1},

	"minecraft:copper_nautilus_armor":    {armor: 4},
	"minecraft:iron_nautilus_armor":      {armor: 5},
	"minecraft:golden_nautilus_armor":    {armor: 7},
	"minecraft:diamond_nautilus_armor":   {armor: 11, toughness: 2},
	"minecraft:netherite_nautilus_armor": {armor: 19, toughness: 3, knockbackResistance: 0.1},
}

// IsEmpty reports whether the slot contains no item.
func (s ItemStack) IsEmpty() bool {
	return s.Count <= 0 || s.ItemID == ""
}

// MaxDurability returns Java Edition's vanilla maximum durability for the
// damageable items GoCraft currently supports. A zero result means the item is
// not damageable.
func MaxDurability(itemID string) int {
	if stats, ok := armorItemStatsByID[itemID]; ok {
		return stats.maxDurability
	}
	switch itemID {
	case "minecraft:bow":
		return 384
	case "minecraft:crossbow":
		return 465
	case "minecraft:trident":
		return 250
	case "minecraft:shield":
		return 336
	case "minecraft:fishing_rod", "minecraft:flint_and_steel", "minecraft:brush":
		return 64
	case "minecraft:shears":
		return 238
	case "minecraft:elytra":
		return 432
	case "minecraft:mace":
		return 500
	case "minecraft:carrot_on_a_stick":
		return 25
	case "minecraft:warped_fungus_on_a_stick":
		return 100
	}
	for material, durability := range map[string]int{
		"wooden": 59, "stone": 131, "iron": 250,
		"golden": 32, "diamond": 1561, "netherite": 2031,
	} {
		prefix := "minecraft:" + material + "_"
		if !strings.HasPrefix(itemID, prefix) {
			continue
		}
		switch strings.TrimPrefix(itemID, prefix) {
		case "sword", "pickaxe", "axe", "shovel", "hoe":
			return durability
		}
	}
	return 0
}

// MaxStackSize returns the stack limit used by the inventory implementation.
func MaxStackSize(itemID string) int {
	if _, ok := armorItemStatsByID[itemID]; ok {
		return 1
	}
	if MaxDurability(itemID) > 0 {
		return 1
	}
	return 64
}

// RemainingDurability returns remaining uses, or zero for non-damageable items.
func (s ItemStack) RemainingDurability() int {
	max := MaxDurability(s.ItemID)
	if max == 0 {
		return 0
	}
	remaining := max - s.Damage
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ApplyDamage consumes durability and reports whether the stack broke.
func (s *ItemStack) ApplyDamage(amount int) bool {
	max := MaxDurability(s.ItemID)
	if max == 0 || amount <= 0 || s.IsEmpty() {
		return false
	}
	s.Damage += amount
	if s.Damage < max {
		return false
	}
	s.Count--
	if s.Count <= 0 {
		*s = ItemStack{}
	} else {
		s.Damage = 0
	}
	return true
}

// ArmorPoints returns the vanilla armour value contributed by an armour item.
func ArmorPoints(itemID string) int {
	return armorItemStatsByID[itemID].armor
}

// ArmorToughness returns the vanilla toughness supplied by one armour piece.
func ArmorToughness(itemID string) float32 {
	return armorItemStatsByID[itemID].toughness
}

// ArmorKnockbackResistance returns the vanilla knockback resistance supplied
// by one armour piece. Netherite contributes 0.1 per equipped piece.
func ArmorKnockbackResistance(itemID string) float32 {
	return armorItemStatsByID[itemID].knockbackResistance
}

// LegacyAttackDamage returns total melee damage for pre-1.9-style combat.
// Newer materials are extended consistently from the old progression.
func LegacyAttackDamage(itemID string) float32 {
	switch itemID {
	case "minecraft:wooden_sword", "minecraft:golden_sword":
		return 4
	case "minecraft:stone_sword":
		return 5
	case "minecraft:iron_sword":
		return 6
	case "minecraft:diamond_sword":
		return 7
	case "minecraft:netherite_sword":
		return 8
	case "minecraft:wooden_axe", "minecraft:golden_axe":
		return 3
	case "minecraft:stone_axe":
		return 4
	case "minecraft:iron_axe":
		return 5
	case "minecraft:diamond_axe":
		return 6
	case "minecraft:netherite_axe":
		return 7
	case "minecraft:trident":
		return 9
	case "minecraft:mace":
		return 6
	default:
		return 1
	}
}

// AttackAttributes returns the 1.21.4 attack damage and speed shown by vanilla
// for a tool or weapon. The bool is false for items without attack modifiers.
func AttackAttributes(itemID string) (damage, speed float32, ok bool) {
	materialDamage := func(material string) float32 {
		switch material {
		case "wooden", "golden":
			return 0
		case "stone":
			return 1
		case "iron":
			return 2
		case "diamond":
			return 3
		case "netherite":
			return 4
		}
		return 0
	}
	for _, material := range []string{"wooden", "stone", "iron", "golden", "diamond", "netherite"} {
		prefix := "minecraft:" + material + "_"
		if !strings.HasPrefix(itemID, prefix) {
			continue
		}
		bonus := materialDamage(material)
		switch strings.TrimPrefix(itemID, prefix) {
		case "sword":
			return 4 + bonus, 1.6, true
		case "shovel":
			return 2.5 + bonus, 1, true
		case "pickaxe":
			return 2 + bonus, 1.2, true
		case "axe":
			damage = map[string]float32{"wooden": 7, "stone": 9, "iron": 9, "golden": 7, "diamond": 9, "netherite": 10}[material]
			speed = map[string]float32{"wooden": 0.8, "stone": 0.8, "iron": 0.9, "golden": 1, "diamond": 1, "netherite": 1}[material]
			return damage, speed, true
		case "hoe":
			speed = map[string]float32{"wooden": 1, "stone": 2, "iron": 3, "golden": 1, "diamond": 4, "netherite": 4}[material]
			return 1, speed, true
		}
	}
	switch itemID {
	case "minecraft:trident":
		return 9, 1.1, true
	case "minecraft:mace":
		return 6, 0.6, true
	}
	return 0, 0, false
}

// BlockUseDamage returns how much durability a successful block-breaking use
// consumes. Swords take two durability when used to break blocks.
func BlockUseDamage(itemID string) int {
	if strings.HasSuffix(itemID, "_sword") {
		return 2
	}
	if MaxDurability(itemID) > 0 {
		return 1
	}
	return 0
}
