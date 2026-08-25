package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
)

// updatePufferState mirrors vanilla's InflateGoal counters. Inflation changes
// immediately to half-puffed, reaches full after 40 ticks, then deflates in
// two delayed stages after a threat leaves.
func updatePufferState(fish *corentity.Entity, threat bool) bool {
	if fish == nil || fish.Type != corentity.TypePufferfish {
		return false
	}
	before := fish.PufferState
	if threat {
		fish.PufferDeflateTicks = 0
		if fish.PufferInflateTicks == 0 {
			fish.PufferInflateTicks = 1
		}
		if fish.PufferState == 0 {
			fish.PufferState = 1
		} else if fish.PufferInflateTicks > 40 && fish.PufferState == 1 {
			fish.PufferState = 2
		}
		fish.PufferInflateTicks++
	} else {
		fish.PufferInflateTicks = 0
		if fish.PufferDeflateTicks == 0 {
			fish.PufferDeflateTicks = 1
		}
		if fish.PufferState == 2 && fish.PufferDeflateTicks > 60 {
			fish.PufferState = 1
		} else if fish.PufferState == 1 && fish.PufferDeflateTicks > 100 {
			fish.PufferState = 0
		}
		fish.PufferDeflateTicks++
	}
	return before != fish.PufferState
}

func pufferfishCalmType(entityType corentity.EntityType) bool {
	switch entityType {
	case corentity.TypeTurtle, corentity.TypeGuardian, corentity.TypeElderGuardian,
		corentity.TypeCod, corentity.TypePufferfish, corentity.TypeSalmon,
		corentity.TypeTropicalFish, corentity.TypeDolphin, corentity.TypeSquid,
		corentity.TypeGlowSquid, corentity.TypeTadpole:
		return true
	default:
		return false
	}
}

func nearPufferfish(fish *corentity.Entity, x, y, z, radius float64) bool {
	return math.Abs(fish.Position.X-x) <= radius &&
		math.Abs(fish.Position.Y-y) <= radius &&
		math.Abs(fish.Position.Z-z) <= radius
}

func (s *Server) pufferfishThreatNearby(fish *corentity.Entity, entities []*corentity.Entity) bool {
	for _, candidate := range s.allPlayerSessions() {
		p := candidate.Player
		if p != nil && !p.Dead && p.Dimension == s.simulationDimension &&
			p.GameMode != player.GameModeCreative && p.GameMode != player.GameModeSpectator &&
			nearPufferfish(fish, p.Position.X, p.Position.Y, p.Position.Z, 2) {
			return true
		}
	}
	for _, candidate := range entities {
		if candidate == nil || candidate == fish || candidate.Dead || pufferfishCalmType(candidate.Type) {
			continue
		}
		if _, living := pumpkinEntitySpawnSettingsByType[string(candidate.Type)]; living &&
			nearPufferfish(fish, candidate.Position.X, candidate.Position.Y, candidate.Position.Z, 2) {
			return true
		}
	}
	return false
}
