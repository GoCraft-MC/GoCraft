package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

const skeletonBowRange = 15.0

// tickSkeletonAvoidance runs the skeleton goals that vanilla prioritises above
// the bow attack: AvoidEntityGoal(Wolf, 6) and FleeSunGoal. Returns true when
// one of them owns the tick, so the archer combat/strafe is skipped.
func (s *Server) tickSkeletonAvoidance(e *corentity.Entity, ai *mobAI) bool {
	if e == nil || ai == nil || s.world == nil {
		return false
	}
	// AvoidEntityGoal(Wolf, 6): sprint directly away from the nearest wolf.
	if wolf := s.closestEntityOfTypes(e, 6, corentity.TypeWolf); wolf != nil {
		dx, dz := e.Position.X-wolf.Position.X, e.Position.Z-wolf.Position.Z
		if distance := math.Hypot(dx, dz); distance > 0.001 {
			flee := spatial.Vec3{X: e.Position.X + dx/distance*8, Y: e.Position.Y, Z: e.Position.Z + dz/distance*8}
			s.navigateMob(e, ai, flee, pumpkinMovementSpeed(e.Type, 1.2))
			return true
		}
	}
	// FleeSunGoal: seek shade while burning in daylight.
	if s.simulationDimension == dimensionOverworld && burnsInDaylight(e.Type) && s.mobInDirectDaylight(e) {
		if shade, ok := s.findShadeNear(e, 12); ok {
			s.navigateMob(e, ai, shade, pumpkinMovementSpeed(e.Type, 1.0))
			return true
		}
	}
	return false
}

// findShadeNear locates the nearest standable, sky-occluded position within the
// given horizontal radius. A position is shaded when the surface above its head
// blocks the sky (the inverse of mobExposedToSky).
func (s *Server) findShadeNear(e *corentity.Entity, radius int) (spatial.Vec3, bool) {
	ex := int(math.Floor(e.Position.X))
	ez := int(math.Floor(e.Position.Z))
	ey := int(math.Floor(e.Position.Y))
	var best spatial.Vec3
	found := false
	bestDist := math.MaxFloat64
	for dx := -radius; dx <= radius; dx++ {
		for dz := -radius; dz <= radius; dz++ {
			x, z := ex+dx, ez+dz
			surfaceY, loaded := s.world.SurfaceYIfLoaded(x, z)
			if !loaded {
				continue
			}
			groundY := s.world.GroundYAtOrBelow(x, z, ey+2)
			if groundY < coreworld.WorldMinY {
				continue
			}
			// Standing here puts the head near groundY+2; shaded if the surface
			// blocks sky above that.
			if surfaceY <= groundY+2 {
				continue
			}
			distance := float64(dx*dx + dz*dz)
			if distance < bestDist {
				bestDist = distance
				best = spatial.Vec3{X: float64(x) + 0.5, Y: float64(groundY + 1), Z: float64(z) + 0.5}
				found = true
			}
		}
	}
	return best, found
}

// tickSkeletonStrafe mirrors RangedBowAttackGoal's strafe: circle the target,
// re-rolling direction every 20 ticks, backing off when too close and closing
// when too far, so the skeleton kites instead of standing still.
func (s *Server) tickSkeletonStrafe(e *corentity.Entity, ai *mobAI, targetPos spatial.Vec3, distance float64) {
	ai.strafeTick++
	if ai.strafeTick >= 20 {
		ai.strafeTick = 0
		if ai.rng.Float64() < 0.3 {
			ai.strafeClockwise = !ai.strafeClockwise
		}
		if ai.rng.Float64() < 0.3 {
			ai.strafeBackwards = !ai.strafeBackwards
		}
	}
	if distance > skeletonBowRange*0.866 {
		ai.strafeBackwards = false
	} else if distance < skeletonBowRange*0.5 {
		ai.strafeBackwards = true
	}
	dx, dz := targetPos.X-e.Position.X, targetPos.Z-e.Position.Z
	if distance < 0.001 {
		e.VX, e.VZ = 0, 0
		return
	}
	ux, uz := dx/distance, dz/distance
	forward := 1.0
	if ai.strafeBackwards {
		forward = -1.0
	}
	side := -1.0
	if ai.strafeClockwise {
		side = 1.0
	}
	mvx := ux*forward - uz*side
	mvz := uz*forward + ux*side
	if mag := math.Hypot(mvx, mvz); mag > 0.001 {
		speed := pumpkinMovementSpeed(e.Type, 1.0)
		e.VX, e.VZ = mvx/mag*speed, mvz/mag*speed
	}
}
