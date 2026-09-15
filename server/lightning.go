package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/java/handler"
)

const (
	lightningLifeTicks    = 10
	lightningDamageRadius = 3.0
	lightningStrikeRadius = 15.0
	lightningFireTicks    = 160
)

// tickWeatherLightning spawns occasional lightning during thunderstorms near
// players, standing in for vanilla's per-chunk thunder roll. Runs once per tick.
func (s *Server) tickWeatherLightning() {
	if s.world == nil || s.game == nil || s.spawnRNG == nil {
		return
	}
	if _, thundering := s.currentWeather(); !thundering {
		return
	}
	for _, sess := range s.allPlayerSessions() {
		p := sess.Player
		if p == nil || p.Dead || p.Dimension != s.simulationDimension {
			continue
		}
		if s.spawnRNG.Intn(6000) != 0 {
			continue
		}
		x := int(math.Floor(p.Position.X)) + s.spawnRNG.Intn(128) - 64
		z := int(math.Floor(p.Position.Z)) + s.spawnRNG.Intn(128) - 64
		surfaceY, loaded := s.world.SurfaceYIfLoaded(x, z)
		if !loaded {
			continue
		}
		s.strikeLightning(float64(x)+0.5, float64(surfaceY), float64(z)+0.5)
	}
}

// strikeLightning spawns a short-lived lightning bolt and applies its effects:
// charging nearby creepers, and burning/damaging entities and players at the
// strike column.
func (s *Server) strikeLightning(x, y, z float64) {
	bolt := corentity.New(s.game.NextEntityID(), newRandomUUID(), corentity.TypeLightningBolt, x, y, z)
	s.world.Entities.Add(bolt)
	handler.BroadcastSpawnMob(bolt, s.sessions)
	handler.BroadcastSoundAt(s.sessions, "minecraft:entity.lightning_bolt.thunder", handler.SoundCategoryAmbient, x, y, z, 10000, 1)
	handler.BroadcastSoundAt(s.sessions, "minecraft:entity.lightning_bolt.impact", handler.SoundCategoryAmbient, x, y, z, 2, 1)

	for _, e := range s.world.Entities.Snapshot() {
		if e == nil || e == bolt || e.Dead || e.Type == corentity.TypeLightningBolt {
			continue
		}
		horizontal := math.Hypot(e.Position.X-x, e.Position.Z-z)
		if horizontal <= lightningStrikeRadius && e.Type == corentity.TypeCreeper && !e.Charged {
			e.Charged = true
			handler.BroadcastMobMetadataInDimension(e, s.sessions, s.simulationDimension)
		}
		if horizontal <= lightningDamageRadius && math.Abs(e.Position.Y-y) <= 6 {
			e.FireTicks = max(e.FireTicks, lightningFireTicks)
			s.world.QueueEntityDamage(e.EntityID, 5)
		}
	}
	for _, sess := range s.allPlayerSessions() {
		p := sess.Player
		if p == nil || p.Dead || p.Dimension != s.simulationDimension {
			continue
		}
		if math.Hypot(p.Position.X-x, p.Position.Z-z) <= lightningDamageRadius && math.Abs(p.Position.Y-y) <= 6 {
			handler.DamagePlayerFromSource(sess, 5, "was struck by lightning", s.sessions, x, z)
		}
	}
}
