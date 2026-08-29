package world

import "strings"

// DoorHinge returns the vanilla Java door hinge for a lower-half placement.
// It mirrors DoorBlock#getHinge: neighbouring doors and full blocks override
// the cursor side, otherwise the hit position relative to the door facing is
// used. Keeping this in the edition-neutral world core gives Java and Bedrock
// identical double-door behaviour.
func DoorHinge(w *World, x, y, z int, facing string, clickX, clickZ float32) string {
	dx, dz := horizontalFacingOffset(facing)
	// Vanilla's counter-clockwise side of the placement direction.
	lx, lz := dz, -dx
	rx, rz := -dz, dx

	left := w.GetBlock(x+lx, y, z+lz)
	leftUpper := w.GetBlock(x+lx, y+1, z+lz)
	right := w.GetBlock(x+rx, y, z+rz)
	rightUpper := w.GetBlock(x+rx, y+1, z+rz)

	score := 0
	if doorHingeFullBlock(left) {
		score--
	}
	if doorHingeFullBlock(leftUpper) {
		score--
	}
	if doorHingeFullBlock(right) {
		score++
	}
	if doorHingeFullBlock(rightUpper) {
		score++
	}

	leftDoor := isLowerDoor(left)
	rightDoor := isLowerDoor(right)
	if (leftDoor && !rightDoor) || score > 0 {
		return "right"
	}
	if (rightDoor && !leftDoor) || score < 0 {
		return "left"
	}

	// Vanilla cursor rule from DoorBlock#getHinge.
	if (dx < 0 && clickZ < 0.5) || (dx > 0 && clickZ > 0.5) ||
		(dz < 0 && clickX > 0.5) || (dz > 0 && clickX < 0.5) {
		return "right"
	}
	return "left"
}

func horizontalFacingOffset(facing string) (dx, dz int) {
	switch facing {
	case "north":
		return 0, -1
	case "south":
		return 0, 1
	case "west":
		return -1, 0
	case "east":
		return 1, 0
	default:
		return 0, 1
	}
}

func isLowerDoor(block Block) bool {
	name := block.ResourceLocation()
	return strings.HasSuffix(name, "_door") && !strings.HasSuffix(name, "_trapdoor") && block.Properties["half"] == "lower"
}

// doorHingeFullBlock is intentionally conservative. Door hinge obstruction
// checks use a full collision cube in vanilla, not merely "non-air".
func doorHingeFullBlock(block Block) bool {
	name := block.ResourceLocation()
	if block.IsAir() || IsFluidBlock(name) ||
		strings.HasSuffix(name, "_slab") || strings.HasSuffix(name, "_stairs") ||
		strings.HasSuffix(name, "_fence") || strings.HasSuffix(name, "_fence_gate") ||
		strings.HasSuffix(name, "_wall") || strings.HasSuffix(name, "_door") ||
		strings.HasSuffix(name, "_trapdoor") || strings.HasSuffix(name, "_button") ||
		strings.HasSuffix(name, "_pressure_plate") || strings.Contains(name, "torch") ||
		strings.Contains(name, "rail") || name == "minecraft:redstone_wire" ||
		strings.HasSuffix(name, "_leaves") || strings.Contains(name, "glass") {
		return false
	}
	return IsSolidLandingSurface(name)
}

// AttachmentSupportUpdate describes a neighbour update caused by a support
// block changing. Previous is retained when a block is removed so adapters can
// evaluate the same vanilla loot table after the world has already become air.
type AttachmentSupportUpdate struct {
	Change   BlockChange
	Previous Block
	Removed  bool
}

// ApplyAttachmentSupportUpdatesAround applies support-dependent neighbour
// updates around a changed block. Bells may switch attachment state instead of
// breaking; other unsupported attachments are removed.
func (w *World) ApplyAttachmentSupportUpdatesAround(x, y, z int) []AttachmentSupportUpdate {
	updates := make([]AttachmentSupportUpdate, 0, 4)
	for _, pos := range neighbors6(x, y, z) {
		block := w.GetBlock(pos[0], pos[1], pos[2])
		if block.ResourceLocation() == "minecraft:bell" {
			updated, survives := bellAttachmentNeighborState(w, pos[0], pos[1], pos[2], block)
			if !survives {
				w.SetBlock(pos[0], pos[1], pos[2], Air)
				updates = append(updates, AttachmentSupportUpdate{
					Change:   BlockChange{X: pos[0], Y: pos[1], Z: pos[2], Block: Air},
					Previous: block,
					Removed:  true,
				})
			} else if updated.Properties["attachment"] != block.Properties["attachment"] ||
				updated.Properties["facing"] != block.Properties["facing"] {
				w.SetBlock(pos[0], pos[1], pos[2], updated)
				updates = append(updates, AttachmentSupportUpdate{
					Change:   BlockChange{X: pos[0], Y: pos[1], Z: pos[2], Block: updated},
					Previous: block,
				})
			}
			continue
		}
		sx, sy, sz, ok := attachmentSupportPosition(pos[0], pos[1], pos[2], block)
		if !ok || sx != x || sy != y || sz != z {
			continue
		}
		if IsSolidLandingSurface(w.GetBlock(sx, sy, sz).ResourceLocation()) {
			continue
		}
		w.SetBlock(pos[0], pos[1], pos[2], Air)
		updates = append(updates, AttachmentSupportUpdate{
			Change:   BlockChange{X: pos[0], Y: pos[1], Z: pos[2], Block: Air},
			Previous: block,
			Removed:  true,
		})
	}
	return updates
}

// BreakUnsupportedAttachmentsAround keeps the original block-change API for
// callers that only need world mutations and client updates.
func (w *World) BreakUnsupportedAttachmentsAround(x, y, z int) []BlockChange {
	updates := w.ApplyAttachmentSupportUpdatesAround(x, y, z)
	changes := make([]BlockChange, 0, len(updates))
	for _, update := range updates {
		changes = append(changes, update.Change)
	}
	return changes
}

func attachmentSupportPosition(x, y, z int, block Block) (sx, sy, sz int, ok bool) {
	return AttachmentSupportPosition(x, y, z, block)
}

// Wall-facing attachments point away from the block they are attached to.
func attachmentWallSupportOffset(facing string) (dx, dz int) {
	switch facing {
	case "north":
		return 0, 1
	case "south":
		return 0, -1
	case "west":
		return 1, 0
	case "east":
		return -1, 0
	default:
		return 0, 0
	}
}

// isRedstonePowerConductor reports full blocks that may become powered and
// relay that power to adjacent redstone components. It is separate from
// IsRedstoneConductor, whose existing meaning is the explicit redstone parts
// (wire/repeater/comparator/block).
func isRedstonePowerConductor(block Block) bool {
	name := block.ResourceLocation()
	if IsRedstoneConductor(name) {
		return true
	}
	if block.IsAir() || IsFluidBlock(name) || IsRedstoneSource(name) || IsRedstoneLoad(name) ||
		strings.HasSuffix(name, "_slab") || strings.HasSuffix(name, "_stairs") ||
		strings.HasSuffix(name, "_fence") || strings.HasSuffix(name, "_fence_gate") ||
		strings.HasSuffix(name, "_wall") || strings.HasSuffix(name, "_door") ||
		strings.HasSuffix(name, "_trapdoor") || strings.HasSuffix(name, "_button") ||
		strings.HasSuffix(name, "_pressure_plate") || strings.Contains(name, "torch") ||
		strings.Contains(name, "rail") || strings.HasSuffix(name, "_leaves") ||
		strings.Contains(name, "glass") {
		return false
	}
	return IsSolidLandingSurface(name)
}
