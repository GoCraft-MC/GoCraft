package world

import "math"

// BellRingDirection validates a hit against a bell and returns the horizontal
// direction used by the transient client animation. Faces use Minecraft's
// common direction indices: down, up, north, south, west, east.
func BellRingDirection(block Block, face int32, hitY float32) (string, bool) {
	if block.ResourceLocation() != "minecraft:bell" || face < 2 || face > 5 || hitY > 0.8124 {
		return "", false
	}
	hitDirection := attachmentFacingForFace(face)
	hitAxis := horizontalAxis(hitDirection)
	facingAxis := horizontalAxis(block.Properties["facing"])

	switch block.Properties["attachment"] {
	case "floor":
		if hitAxis != facingAxis {
			return "", false
		}
	case "single_wall", "double_wall":
		if hitAxis == facingAxis {
			return "", false
		}
	case "ceiling":
		// Every horizontal face may ring a ceiling bell.
	default:
		return "", false
	}
	return hitDirection, true
}

// BellFacingDirection returns a valid horizontal bell direction for non-hit
// activations such as a redstone rising edge.
func BellFacingDirection(block Block) string {
	switch block.Properties["facing"] {
	case "north", "south", "west", "east":
		return block.Properties["facing"]
	default:
		return "north"
	}
}

// BellProjectileFace derives the face entered by a projectile from its motion.
func BellProjectileFace(dx, dy, dz float64) int32 {
	ax, ay, az := math.Abs(dx), math.Abs(dy), math.Abs(dz)
	if ay >= ax && ay >= az {
		if dy > 0 {
			return 0
		}
		return 1
	}
	if ax >= az {
		if dx > 0 {
			return 4
		}
		return 5
	}
	if dz > 0 {
		return 2
	}
	return 3
}

func horizontalAxis(direction string) byte {
	if direction == "east" || direction == "west" {
		return 'x'
	}
	return 'z'
}
