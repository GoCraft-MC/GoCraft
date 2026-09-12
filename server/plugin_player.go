package server

import (
	"math"

	"GoCraft/core/player"
)

func (s *Server) installPlayerEvents(p *player.Player) {
	p.BeforeDamage = func(amount float32, cause string) (float32, bool) {
		damage := float64(amount)
		if s.plugins != nil && !s.plugins.EmitPlayerDamage(p, &damage, cause) {
			return 0, false
		}
		return float32(damage), damage > 0 && damage <= math.MaxFloat32
	}
	p.OnDeath = func(dead *player.Player) {
		// The health pipeline has already allowed a held totem to prevent death.
		if s.plugins != nil {
			s.plugins.EmitPlayerDeath(dead, dead.LastDamageCause)
		}
		s.dropPlayerInventory(dead)
	}
}

func (s *Server) unregisterPlayer(uuid [16]byte, reason string) {
	if p := s.game.RemovePlayer(uuid); p != nil && s.plugins != nil {
		s.plugins.EmitPlayerQuit(p, reason)
	}
}

func inventoryEventContainer(p *player.Player) string {
	if p.OpenContainerKind != "" {
		return p.OpenContainerKind
	}
	return "minecraft:inventory"
}
