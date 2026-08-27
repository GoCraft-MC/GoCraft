package server

import (
	"fmt"
	"math"

	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func spreadLoadedPositions(w *coreworld.World, centerX, centerZ, distance, radius float64, count int, random func() float64) ([]spatial.Vec3, error) {
	if w == nil || count < 1 || distance < 0 || radius < distance {
		return nil, fmt.Errorf("maximum range must be at least the spread distance")
	}
	positions := make([]spatial.Vec3, 0, count)
	for attempts := 0; len(positions) < count && attempts < count*1000; attempts++ {
		x := int(math.Floor(centerX + (random()*2-1)*radius))
		z := int(math.Floor(centerZ + (random()*2-1)*radius))
		y, loaded := w.SurfaceYIfLoaded(x, z)
		if !loaded || !safeSpreadSupport(w.GetBlock(x, y, z).ResourceLocation()) {
			continue
		}
		candidate := spatial.Vec3{X: float64(x) + 0.5, Y: float64(y + 1), Z: float64(z) + 0.5}
		if free, loaded := w.CanEntityOccupyIfLoaded(candidate.X, candidate.Y, candidate.Z); !loaded || !free {
			continue
		}
		separated := true
		for _, other := range positions {
			dx, dz := candidate.X-other.X, candidate.Z-other.Z
			if dx*dx+dz*dz < distance*distance {
				separated = false
				break
			}
		}
		if separated {
			positions = append(positions, candidate)
		}
	}
	if len(positions) != count {
		return nil, fmt.Errorf("could not find %d safe loaded positions", count)
	}
	return positions, nil
}

func safeSpreadSupport(name string) bool {
	return coreworld.IsEntitySupportBlock(name) && name != "minecraft:fire" &&
		name != "minecraft:lava" && name != "minecraft:water" && name != "minecraft:magma_block"
}
