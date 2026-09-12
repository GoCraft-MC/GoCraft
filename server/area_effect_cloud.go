package server

import (
	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
)

const areaEffectCloudWarmup = 10

// tickAreaEffectCloud advances and applies a lingering potion cloud. It
// returns true once the cloud should be removed from the world.
func (s *Server) tickAreaEffectCloud(cloud *corentity.Entity) bool {
	if cloud == nil || cloud.AgeTicks >= cloud.CloudDurationTicks {
		return true
	}
	if cloud.AgeTicks < areaEffectCloudWarmup {
		return false
	}
	cloud.CloudRadius += cloud.CloudRadiusGrowth
	if cloud.CloudRadius < 0.5 {
		return true
	}
	if cloud.AgeTicks%10 != 0 {
		return false
	}
	outcome, ok := player.PotionOutcomeFor(cloud.ProjectileItem)
	if !ok {
		return false
	}
	for _, target := range s.allPlayerSessions() {
		p := target.Player
		if p == nil || p.Dead {
			continue
		}
		if expiry, found := cloud.CloudTargets[p.EntityID]; found && cloud.AgeTicks < expiry {
			continue
		}
		dx, dz := p.Position.X-cloud.Position.X, p.Position.Z-cloud.Position.Z
		if dx*dx+dz*dz > cloud.CloudRadius*cloud.CloudRadius {
			continue
		}
		s.applyPotionOutcome(target, outcome, 1, 0.25)
		cloud.CloudTargets[p.EntityID] = cloud.AgeTicks + cloud.CloudReapplicationDelay
		cloud.CloudRadius += cloud.CloudRadiusOnUse
		if cloud.CloudRadius <= 0.5 {
			return true
		}
	}
	return false
}
