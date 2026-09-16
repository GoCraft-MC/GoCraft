package server

import (
	"math"
	"math/rand"

	corentity "GoCraft/core/entity"
)

func isCubeMob(t corentity.EntityType) bool {
	return t == corentity.TypeSlime || t == corentity.TypeMagmaCube
}

// setSlimeSize applies a cube-mob size (1, 2 or 4): max health is size squared.
// It does not broadcast; callers send a spawn packet or a metadata update.
func setSlimeSize(e *corentity.Entity, size int32) {
	if size < 1 {
		size = 1
	}
	e.SlimeSize = size
	e.MaxHealth = float32(size * size)
	e.Health = e.MaxHealth
}

// randomSlimeSize returns a natural cube-mob size: 1, 2 or 4 (1 << rng(3)).
func randomSlimeSize(rng *rand.Rand) int32 {
	if rng == nil {
		return 1
	}
	return int32(1) << uint(rng.Intn(3))
}

// sizeNaturalCubeMob assigns a random size to a freshly created slime/magma cube
// before it is broadcast, so it spawns at a natural 1/2/4 size.
func (s *Server) sizeNaturalCubeMob(e *corentity.Entity) {
	if e == nil || !isCubeMob(e.Type) {
		return
	}
	setSlimeSize(e, randomSlimeSize(s.spawnRNG))
}

// splitSlimeOnDeath spawns 2-4 half-size cubes when a size>1 slime or magma cube
// dies, mirroring AbstractCubeMob.remove. Returns the spawned children.
func (s *Server) splitSlimeOnDeath(e *corentity.Entity) []*corentity.Entity {
	if e == nil || s.world == nil || s.game == nil || !isCubeMob(e.Type) || e.SlimeSize <= 1 {
		return nil
	}
	halfSize := e.SlimeSize / 2
	count := 2 + s.spawnRNG.Intn(3)
	children := make([]*corentity.Entity, 0, count)
	for i := 0; i < count; i++ {
		angle := float64(i) / float64(count) * 2 * math.Pi
		child := corentity.New(s.game.NextEntityID(), newRandomUUID(), e.Type,
			e.Position.X+math.Cos(angle)*0.5, e.Position.Y, e.Position.Z+math.Sin(angle)*0.5)
		setSlimeSize(child, halfSize)
		// Inherit the parent's natural-spawn status so the children participate
		// in the same despawn lifecycle instead of leaking forever.
		child.NaturalSpawned = e.NaturalSpawned
		child.OnGround = e.OnGround
		s.world.Entities.Add(child)
		// Do not broadcast here: the caller appends the returned children to the
		// tick's spawned batch, which sends their spawn packets once in the owning
		// dimension. Broadcasting again here double-sends and leaks across dimensions.
		children = append(children, child)
	}
	return children
}
