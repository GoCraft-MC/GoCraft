package bedrock

import (
	"GoCraft/core/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func (l *Listener) syncLocalStatusEffects(viewer *bedrockSession, tick uint64) {
	p := l.game.GetPlayer(viewer.uuid)
	if p == nil {
		return
	}
	current := p.StatusEffectsSnapshot()
	previous := make(map[string]player.StatusEffect, len(viewer.lastStatusEffects))
	for _, effect := range viewer.lastStatusEffects {
		previous[effect.ID] = effect
	}
	present := make(map[string]struct{}, len(current))
	for _, effect := range current {
		present[effect.ID] = struct{}{}
		effectType := EffectType(effect.ID)
		before, found := previous[effect.ID]
		operation := statusEffectOperation(before, found, effect)
		if effectType == 0 || operation == 0 {
			continue
		}
		_ = viewer.conn.WritePacket(&packet.MobEffect{
			EntityRuntimeID: bedrockSelfRuntimeID, Operation: operation,
			EffectType: effectType, Amplifier: effect.Amplifier,
			Particles: effect.ShowParticles, Duration: effect.Duration,
			Tick: tick, Ambient: effect.Ambient,
		})
	}
	for _, effect := range viewer.lastStatusEffects {
		if _, found := present[effect.ID]; found || EffectType(effect.ID) == 0 {
			continue
		}
		_ = viewer.conn.WritePacket(&packet.MobEffect{
			EntityRuntimeID: bedrockSelfRuntimeID, Operation: packet.MobEffectRemove,
			EffectType: EffectType(effect.ID), Tick: tick,
		})
	}
	viewer.lastStatusEffects = current
}

func statusEffectOperation(before player.StatusEffect, found bool, current player.StatusEffect) byte {
	if !found {
		return packet.MobEffectAdd
	}
	if before.Amplifier != current.Amplifier || before.Ambient != current.Ambient ||
		before.ShowParticles != current.ShowParticles || current.Duration > before.Duration-1 {
		return packet.MobEffectModify
	}
	return 0
}
