package player

var exclusiveEnchantments = map[string]map[string]bool{
	"armor":    setOf("protection", "blast_protection", "fire_protection", "projectile_protection"),
	"boots":    setOf("frost_walker", "depth_strider"),
	"bow":      setOf("infinity", "mending"),
	"crossbow": setOf("multishot", "piercing"),
	"damage":   setOf("sharpness", "smite", "bane_of_arthropods", "impaling", "density", "breach"),
	"mining":   setOf("fortune", "silk_touch"),
	"riptide":  setOf("loyalty", "channeling"),
}

func setOf(ids ...string) map[string]bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set["minecraft:"+id] = true
	}
	return set
}

func (e Enchantment) CompatibleWith(stack ItemStack) bool {
	for _, applied := range stack.EnchantmentLevels() {
		other, ok := EnchantmentByID(applied.ID)
		if !ok || other.ID == e.ID || enchantmentsConflict(e, other) {
			return false
		}
	}
	return true
}

func enchantmentsConflict(first, second Enchantment) bool {
	if first.ExclusiveSet != "" && exclusiveEnchantments[first.ExclusiveSet][second.ID] {
		return true
	}
	return second.ExclusiveSet != "" && exclusiveEnchantments[second.ExclusiveSet][first.ID]
}
