package player

// ApplyConsumableCleansing removes the effects cleared by a finished vanilla
// consumable and returns them for adapter removal packets.
func (p *Player) ApplyConsumableCleansing(itemID string) []StatusEffect {
	if p == nil {
		return nil
	}
	switch itemID {
	case "minecraft:honey_bottle":
		if removed, ok := p.RemoveStatusEffect("minecraft:poison"); ok {
			return []StatusEffect{removed}
		}
	case "minecraft:milk_bucket":
		return p.ClearStatusEffects()
	}
	return nil
}
