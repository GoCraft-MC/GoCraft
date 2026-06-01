package world

import (
	"math"

	"GoCraft/core/spatial"
)

// EnsureSafeArrival validates a dimension-change destination before the
// player's dimension is switched. If no natural two-block-high opening is
// available in the column, it creates a small obsidian landing platform and
// clears a three-block-high chamber at a dimension-appropriate height.
func (w *World) EnsureSafeArrival(target spatial.Vec3, dimension int32) spatial.Vec3 {
	if w == nil {
		return target
	}
	x, z := int(math.Floor(target.X)), int(math.Floor(target.Z))
	minimumY, maximumY := WorldMinY+2, WorldMaxY-2
	preferredY := int(math.Floor(target.Y))
	if dimension == 1 {
		minimumY, maximumY = 8, 118
		if preferredY < minimumY || preferredY > maximumY {
			preferredY = 64
		}
	}
	preferredY = max(minimumY, min(maximumY, preferredY))

	if safeArrivalAt(w, x, preferredY, z) {
		return spatial.Vec3{X: float64(x) + 0.5, Y: float64(preferredY), Z: float64(z) + 0.5}
	}
	searchRadius := max(preferredY-minimumY, maximumY-preferredY)
	for offset := 1; offset <= searchRadius; offset++ {
		for _, y := range [...]int{preferredY + offset, preferredY - offset} {
			if y < minimumY || y > maximumY || !safeArrivalAt(w, x, y, z) {
				continue
			}
			return spatial.Vec3{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5}
		}
	}

	fallbackY := preferredY
	if dimension == 1 {
		fallbackY = 64
	} else if dimension == 2 {
		fallbackY = 49
	}
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			w.SetBlock(x+dx, fallbackY-1, z+dz, block("obsidian"))
		}
	}
	for y := fallbackY; y <= fallbackY+2; y++ {
		w.SetBlock(x, y, z, Air)
	}
	return spatial.Vec3{X: float64(x) + 0.5, Y: float64(fallbackY), Z: float64(z) + 0.5}
}

func safeArrivalAt(w *World, x, y, z int) bool {
	feet := w.GetBlock(x, y, z).ResourceLocation()
	head := w.GetBlock(x, y+1, z).ResourceLocation()
	below := w.GetBlock(x, y-1, z).ResourceLocation()
	return !IsEntitySupportBlock(feet) && !IsEntitySupportBlock(head) &&
		!hazardousArrivalBlock(feet) && !hazardousArrivalBlock(head) &&
		IsEntitySupportBlock(below) && !IsFluidBlock(below) && !hazardousArrivalSupport(below)
}

func hazardousArrivalBlock(name string) bool {
	switch name {
	case "minecraft:water", "minecraft:lava", "minecraft:fire", "minecraft:soul_fire",
		"minecraft:powder_snow", "minecraft:cactus", "minecraft:sweet_berry_bush":
		return true
	default:
		return false
	}
}

func hazardousArrivalSupport(name string) bool {
	switch name {
	case "minecraft:magma_block", "minecraft:campfire", "minecraft:soul_campfire", "minecraft:cactus":
		return true
	default:
		return false
	}
}
