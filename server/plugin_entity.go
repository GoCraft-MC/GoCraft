package server

import (
	"math"

	"GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func (s *Server) installWorldEvents(w *coreworld.World, dimension int32) {
	if w == nil {
		return
	}
	w.BeforeEntityDamage = func(target *entity.Entity, amount float32) (float32, bool) {
		// The existing queue aggregates hits. Publish each before aggregation,
		// so cancellation also prevents player weapon wear at the producer.
		return s.filterEntityDamage(target, amount, "queued", dimension)
	}
}

func (s *Server) filterEntityDamage(target *entity.Entity, amount float32, cause string, dimension int32) (float32, bool) {
	if target == nil || target.Dead {
		return 0, false
	}
	damage := float64(amount)
	if s.plugins != nil && !s.plugins.EmitEntityDamage(int64(target.EntityID), string(target.Type), &damage, cause, int64(dimension)) {
		return 0, false
	}
	return float32(damage), damage > 0 && damage <= math.MaxFloat32
}

func (s *Server) damageEnvironmentalEntity(target *entity.Entity, amount float32, cause string) bool {
	damage, allowed := s.filterEntityDamage(target, amount, cause, s.simulationDimension)
	if allowed {
		target.Damage(damage)
	}
	return allowed
}
