package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
)

// parityMobHostileToPlayer applies target predicates that are lost when every
// monster is routed through the legacy nearest-player controller. It is used by
// both navigation and line-of-sight attack gates, so a neutral mob neither
// chases nor damages a player while its vanilla target predicate is false.
func (s *Server) parityMobHostileToPlayer(e *corentity.Entity, target *player.Player) bool {
	if e == nil || target == nil || target.Dead || target.GameMode == player.GameModeCreative || target.GameMode == player.GameModeSpectator {
		return false
	}
	provoked := target.LastAttackedEntityID == e.EntityID
	state := parityState(e)
	if provoked {
		switch e.Type {
		case corentity.TypeSpider, corentity.TypeCaveSpider, corentity.TypePiglin,
			corentity.TypeEnderman, corentity.TypeWarden, corentity.TypeCreaker,
			corentity.TypeZombifiedPiglin:
			state.angerTicks = 400
		}
	}

	switch e.Type {
	case corentity.TypePiglin:
		// Adult piglins tolerate players wearing any gold armour unless that
		// player attacked them. Piglin brutes intentionally do not use this rule.
		return provoked || state.angerTicks > 0 || !playerWearsGoldArmor(target)
	case corentity.TypeSpider, corentity.TypeCaveSpider:
		// Spiders become neutral at high skylight unless already angered.
		return provoked || state.angerTicks > 0 || !s.mobInDirectDaylight(e)
	case corentity.TypeEnderman:
		return provoked || state.angerTicks > 0 || isPlayerStaringAtEnderman(target, e)
	case corentity.TypeCreaker:
		// Creakings freeze while a survival/adventure player is looking at them.
		return provoked || !isPlayerLookingAtMob(target, e, 0.045)
	case corentity.TypeWarden:
		// The Warden attacks from anger, not nearest-player aggro. Direct attacks
		// are a high-priority anger source; vibration-driven anger is maintained by
		// the dedicated parity state when available.
		return provoked || state.angerTicks > 0
	}
	return true
}

func playerWearsGoldArmor(p *player.Player) bool {
	if p == nil {
		return false
	}
	// Java inventory slots 5..8 are helmet/chest/legs/boots in GoCraft's
	// canonical inventory layout.
	for slot := 5; slot <= 8 && slot < len(p.Inventory); slot++ {
		switch p.Inventory[slot].ItemID {
		case "minecraft:golden_helmet", "minecraft:golden_chestplate", "minecraft:golden_leggings", "minecraft:golden_boots":
			return true
		}
	}
	return false
}

func isPlayerLookingAtMob(p *player.Player, e *corentity.Entity, angularSlack float64) bool {
	if p == nil || e == nil {
		return false
	}
	yawRad := -float64(p.Rotation.Yaw) * math.Pi / 180
	pitchRad := float64(p.Rotation.Pitch) * math.Pi / 180
	cosPitch := math.Cos(pitchRad)
	lookX := math.Sin(yawRad) * cosPitch
	lookY := -math.Sin(pitchRad)
	lookZ := math.Cos(yawRad) * cosPitch

	dx := e.Position.X - p.Position.X
	dy := e.Position.Y + 1.0 - (p.Position.Y + 1.62)
	dz := e.Position.Z - p.Position.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance < 0.1 {
		return true
	}
	dot := lookX*(dx/distance) + lookY*(dy/distance) + lookZ*(dz/distance)
	threshold := 1.0 - angularSlack/distance
	return dot > threshold
}

// refreshParityProvocation is called serially before parallel passive work. It
// handles neutral mobs that are outside the old hostile/passive classifiers and
// propagates group anger where vanilla does so.
func (s *Server) refreshParityProvocation(e *corentity.Entity) {
	if e == nil || s.game == nil {
		return
	}
	var aggressor *player.Player
	s.game.OnlinePlayers(func(candidate *player.Player) {
		if aggressor == nil && candidate.Dimension == s.simulationDimension && candidate.LastAttackedEntityID == e.EntityID {
			aggressor = candidate
		}
	})
	if aggressor == nil {
		return
	}
	state := parityState(e)
	state.angerTicks = 400
	if e.Type != corentity.TypeZombifiedPiglin || s.world == nil {
		return
	}
	// Zombified piglin anger spreads to nearby group members.
	for _, other := range s.world.Entities.Snapshot() {
		if other != nil && !other.Dead && other.Type == corentity.TypeZombifiedPiglin && distance2D(other.Position, e.Position) <= 32 {
			parityState(other).angerTicks = 400
		}
	}
}
