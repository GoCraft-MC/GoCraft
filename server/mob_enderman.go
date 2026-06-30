package server

import (
	"math/rand"
)

// EndermanTeleportRadius is the max block radius for a damage-triggered teleport.
const EndermanTeleportRadius = 32

// TryEndermanTeleport attempts to teleport the enderman away from its attacker.
// Returns the new position if a valid landing spot was found.
func TryEndermanTeleport(ex, ey, ez float64) (nx, ny, nz float64, ok bool) {
	for attempt := 0; attempt < 16; attempt++ {
		dx := (rand.Float64()*2 - 1) * EndermanTeleportRadius
		dz := (rand.Float64()*2 - 1) * EndermanTeleportRadius
		nx = ex + dx
		nz = ez + dz
		ny = ey
		// TODO: validate landing spot against world block data
		return nx, ny, nz, true
	}
	return ex, ey, ez, false
}
