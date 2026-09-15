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
		converted := false
		if horizontal <= lightningStrikeRadius {
			converted = s.entityThunderHit(e)
		}
		if !converted && horizontal <= lightningDamageRadius && math.Abs(e.Position.Y-y) <= 6 {
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

// entityThunderHit applies the per-species vanilla thunderHit effect: charge
// creepers, convert pigs to zombified piglins and villagers to witches (outside
// peaceful), and toggle a mooshroom's variant. Returns true when the entity was
// converted (and thus removed), so the caller skips further damage on it.
func (s *Server) entityThunderHit(e *corentity.Entity) bool {
	switch e.Type {
	case corentity.TypeCreeper:
		if !e.Charged {
			e.Charged = true
			handler.BroadcastMobMetadataInDimension(e, s.sessions, s.simulationDimension)
		}
	case corentity.TypePig:
		if s.currentDifficulty() != 0 { // not peaceful
			s.lightningConvert(e, corentity.TypeZombifiedPiglin, func(n *corentity.Entity) {
				n.MainHandItemID = "minecraft:golden_sword"
			})
			return true
		}
	case corentity.TypeVillager:
		if s.currentDifficulty() != 0 {
			s.lightningConvert(e, corentity.TypeWitch, nil)
			return true
		}
	case corentity.TypeMooshroom:
		// Toggle red<->brown. The client-facing variant metadata is deferred
		// pending confirmation of the 1.21.4 serializer, so this is server-side
		// state plus the convert sound.
		e.MooshroomBrown = !e.MooshroomBrown
		handler.BroadcastSoundAt(s.sessions, "minecraft:entity.mooshroom.convert", handler.SoundCategoryNeutral,
			e.Position.X, e.Position.Y, e.Position.Z, 2, 1)
	}
	return false
}

// lightningConvert replaces an entity with a new one of the target type at the
// same position, mirroring vanilla convertTo: the new entity is spawned and the
// old one removed.
func (s *Server) lightningConvert(old *corentity.Entity, newType corentity.EntityType, setup func(*corentity.Entity)) {
	n := corentity.New(s.game.NextEntityID(), newRandomUUID(), newType, old.Position.X, old.Position.Y, old.Position.Z)
	n.Yaw, n.Pitch = old.Yaw, old.Pitch
	n.IsBaby = old.IsBaby
	if setup != nil {
		setup(n)
	}
	s.world.Entities.Add(n)
	handler.BroadcastSpawnMob(n, s.sessions)

	old.Dead = true
	s.world.Entities.Remove(old.EntityID)
	handler.BroadcastRemoveEntity(old.EntityID, s.sessions)
	delete(s.mobAIs, old.EntityID)
}
