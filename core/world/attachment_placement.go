package world

import "strings"

// IsAttachmentPlacementItem reports block-items whose vanilla placement state
// depends on the clicked support face instead of only the item identifier.
func IsAttachmentPlacementItem(name string) bool {
	if isStandingSignName(name) || isCeilingHangingSignName(name) || isStandingBannerName(name) ||
		isFloorHeadOrSkull(name) || isFloorCoralFan(name) || IsRailBlock(name) {
		return true
	}
	switch name {
	case "minecraft:ladder", "minecraft:lantern", "minecraft:soul_lantern", "minecraft:bell",
		"minecraft:tripwire_hook", "minecraft:flower_pot", "minecraft:candle":
		return true
	}
	return strings.HasSuffix(name, "_candle") && !strings.HasSuffix(name, "_candle_cake")
}

// AttachmentPlacementState applies the shared vanilla placement rules for
// face-attached and support-dependent decorative blocks. face uses Minecraft's
// common direction indices: 0=down, 1=up, 2=north, 3=south, 4=west, 5=east.
// rotation is the 0-15 horizontal rotation used by signs, banners and skulls.
func AttachmentPlacementState(w *World, block Block, x, y, z int, face int32, rotation int, waterlogged bool) (Block, bool, bool) {
	name := block.ResourceLocation()
	if !IsAttachmentPlacementItem(name) {
		return block, false, true
	}
	rotation &= 15

	if isCeilingHangingSignName(name) {
		switch {
		case face == 0:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Properties = map[string]string{
				"attached": "false", "rotation": itoaSmall(rotation), "waterlogged": attachmentBoolString(waterlogged),
			}
			return block, true, true
		case face >= 2 && face <= 5:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Name = strings.TrimSuffix(block.Name, "_hanging_sign") + "_wall_hanging_sign"
			block.Properties = map[string]string{"facing": attachmentFacingForFace(face), "waterlogged": attachmentBoolString(waterlogged)}
			return block, true, true
		default:
			return Block{}, true, false
		}
	}

	if isStandingSignName(name) {
		switch {
		case face == 1:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Properties = map[string]string{"rotation": itoaSmall(rotation), "waterlogged": attachmentBoolString(waterlogged)}
			return block, true, true
		case face >= 2 && face <= 5:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Name = strings.TrimSuffix(block.Name, "_sign") + "_wall_sign"
			block.Properties = map[string]string{"facing": attachmentFacingForFace(face), "waterlogged": attachmentBoolString(waterlogged)}
			return block, true, true
		default:
			return Block{}, true, false
		}
	}

	if isStandingBannerName(name) {
		switch {
		case face == 1:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Properties = map[string]string{"rotation": itoaSmall(rotation)}
			return block, true, true
		case face >= 2 && face <= 5:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Name = strings.TrimSuffix(block.Name, "_banner") + "_wall_banner"
			block.Properties = map[string]string{"facing": attachmentFacingForFace(face)}
			return block, true, true
		default:
			return Block{}, true, false
		}
	}

	if isFloorHeadOrSkull(name) {
		switch {
		case face == 1:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Properties = map[string]string{"rotation": itoaSmall(rotation)}
			return block, true, true
		case face >= 2 && face <= 5:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Name = wallHeadOrSkullName(block.Name)
			block.Properties = map[string]string{"facing": attachmentFacingForFace(face)}
			return block, true, true
		default:
			return Block{}, true, false
		}
	}

	if isFloorCoralFan(name) {
		switch {
		case face == 1:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Properties = map[string]string{"waterlogged": attachmentBoolString(waterlogged)}
			return block, true, true
		case face >= 2 && face <= 5:
			if !attachmentPlacementSupport(w, x, y, z, face) {
				return Block{}, true, false
			}
			block.Name = strings.TrimSuffix(block.Name, "_coral_fan") + "_coral_wall_fan"
			block.Properties = map[string]string{"facing": attachmentFacingForFace(face), "waterlogged": attachmentBoolString(waterlogged)}
			return block, true, true
		default:
			return Block{}, true, false
		}
	}

	switch name {
	case "minecraft:bell":
		placed, valid := bellPlacementState(w, block, x, y, z, face, rotation)
		if !valid {
			return Block{}, true, false
		}
		return placed, true, true

	case "minecraft:ladder":
		if face < 2 || face > 5 || !attachmentPlacementSupport(w, x, y, z, face) {
			return Block{}, true, false
		}
		block.Properties = map[string]string{"facing": attachmentFacingForFace(face), "waterlogged": attachmentBoolString(waterlogged)}
		return block, true, true

	case "minecraft:lantern", "minecraft:soul_lantern":
		hanging := false
		if face == 0 {
			if !attachmentPlacementSupport(w, x, y, z, 0) {
				return Block{}, true, false
			}
			hanging = true
		} else if attachmentPlacementSupportAt(w, x, y-1, z) {
			hanging = false
		} else if attachmentPlacementSupportAt(w, x, y+1, z) {
			hanging = true
		} else {
			return Block{}, true, false
		}
		block.Properties = map[string]string{"hanging": attachmentBoolString(hanging), "waterlogged": attachmentBoolString(waterlogged)}
		return block, true, true

	case "minecraft:tripwire_hook":
		if face < 2 || face > 5 || !attachmentPlacementSupport(w, x, y, z, face) {
			return Block{}, true, false
		}
		block.Properties = map[string]string{"attached": "false", "facing": attachmentFacingForFace(face), "powered": "false"}
		return block, true, true

	case "minecraft:flower_pot":
		if !attachmentPlacementSupportAt(w, x, y-1, z) {
			return Block{}, true, false
		}
		block.Properties = nil
		return block, true, true

	case "minecraft:candle":
		if !attachmentPlacementSupportAt(w, x, y-1, z) {
			return Block{}, true, false
		}
		block.Properties = map[string]string{"candles": "1", "lit": "false", "waterlogged": attachmentBoolString(waterlogged)}
		return block, true, true
	}

	if strings.HasSuffix(name, "_candle") && !strings.HasSuffix(name, "_candle_cake") {
		if !attachmentPlacementSupportAt(w, x, y-1, z) {
			return Block{}, true, false
		}
		block.Properties = map[string]string{"candles": "1", "lit": "false", "waterlogged": attachmentBoolString(waterlogged)}
		return block, true, true
	}

	if IsRailBlock(name) {
		if !attachmentPlacementSupportAt(w, x, y-1, z) {
			return Block{}, true, false
		}
		block.Properties = map[string]string{"shape": "north_south"}
		if name != "minecraft:rail" {
			block.Properties["powered"] = "false"
		}
		return block, true, true
	}

	return block, true, true
}

// AttachmentSupportPosition resolves the exact support block for a placed
// support-dependent block. It is shared by neighbour physics for both editions.
func AttachmentSupportPosition(x, y, z int, block Block) (sx, sy, sz int, ok bool) {
	name := block.ResourceLocation()
	if name == "minecraft:lever" || strings.HasSuffix(name, "_button") {
		switch block.Properties["face"] {
		case "floor":
			return x, y - 1, z, true
		case "ceiling":
			return x, y + 1, z, true
		default:
			dx, dz := attachmentWallSupportOffset(block.Properties["facing"])
			return x + dx, y, z + dz, true
		}
	}

	if name == "minecraft:redstone_wall_torch" || name == "minecraft:wall_torch" ||
		name == "minecraft:soul_wall_torch" || name == "minecraft:tripwire_hook" ||
		name == "minecraft:ladder" || isWallSignName(name) || isWallHangingSignName(name) ||
		isWallBannerName(name) || isWallHeadOrSkull(name) || isWallCoralFan(name) {
		dx, dz := attachmentWallSupportOffset(block.Properties["facing"])
		return x + dx, y, z + dz, true
	}

	if name == "minecraft:lantern" || name == "minecraft:soul_lantern" {
		if block.Properties["hanging"] == "true" {
			return x, y + 1, z, true
		}
		return x, y - 1, z, true
	}

	if isCeilingHangingSignName(name) {
		return x, y + 1, z, true
	}

	if name == "minecraft:torch" || name == "minecraft:soul_torch" || name == "minecraft:redstone_torch" ||
		name == "minecraft:redstone_wire" || name == "minecraft:repeater" || name == "minecraft:comparator" ||
		IsRailBlock(name) || strings.HasSuffix(name, "_pressure_plate") || isStandingSignName(name) ||
		isStandingBannerName(name) || isFloorHeadOrSkull(name) || isFloorCoralFan(name) ||
		name == "minecraft:flower_pot" || name == "minecraft:candle" ||
		(strings.HasSuffix(name, "_candle") && !strings.HasSuffix(name, "_candle_cake")) {
		return x, y - 1, z, true
	}
	return 0, 0, 0, false
}

func attachmentPlacementSupport(w *World, x, y, z int, face int32) bool {
	dx, dy, dz := attachmentFaceOffset(face)
	return attachmentPlacementSupportAt(w, x-dx, y-dy, z-dz)
}

func attachmentPlacementSupportAt(w *World, x, y, z int) bool {
	if w == nil || y < WorldMinY || y > WorldMaxY {
		return false
	}
	return IsSolidLandingSurface(w.GetBlock(x, y, z).ResourceLocation())
}

func attachmentFaceOffset(face int32) (x, y, z int) {
	switch face {
	case 0:
		return 0, -1, 0
	case 1:
		return 0, 1, 0
	case 2:
		return 0, 0, -1
	case 3:
		return 0, 0, 1
	case 4:
		return -1, 0, 0
	case 5:
		return 1, 0, 0
	default:
		return 0, 0, 0
	}
}

func attachmentFacingForFace(face int32) string {
	switch face {
	case 2:
		return "north"
	case 3:
		return "south"
	case 4:
		return "west"
	case 5:
		return "east"
	default:
		return "north"
	}
}

func isStandingSignName(name string) bool {
	return strings.HasSuffix(name, "_sign") && !strings.HasSuffix(name, "_wall_sign") &&
		!strings.HasSuffix(name, "_hanging_sign") && !strings.HasSuffix(name, "_wall_hanging_sign")
}

func isWallSignName(name string) bool {
	return strings.HasSuffix(name, "_wall_sign") && !strings.HasSuffix(name, "_wall_hanging_sign")
}

func isCeilingHangingSignName(name string) bool {
	return strings.HasSuffix(name, "_hanging_sign") && !strings.HasSuffix(name, "_wall_hanging_sign")
}

func isWallHangingSignName(name string) bool { return strings.HasSuffix(name, "_wall_hanging_sign") }
func isStandingBannerName(name string) bool {
	return strings.HasSuffix(name, "_banner") && !strings.HasSuffix(name, "_wall_banner")
}
func isWallBannerName(name string) bool { return strings.HasSuffix(name, "_wall_banner") }
func isFloorCoralFan(name string) bool {
	return strings.HasSuffix(name, "_coral_fan") && !strings.HasSuffix(name, "_coral_wall_fan")
}
func isWallCoralFan(name string) bool { return strings.HasSuffix(name, "_coral_wall_fan") }

func isFloorHeadOrSkull(name string) bool {
	switch name {
	case "minecraft:skeleton_skull", "minecraft:wither_skeleton_skull", "minecraft:zombie_head",
		"minecraft:player_head", "minecraft:creeper_head", "minecraft:dragon_head", "minecraft:piglin_head":
		return true
	default:
		return false
	}
}

func isWallHeadOrSkull(name string) bool {
	switch name {
	case "minecraft:skeleton_wall_skull", "minecraft:wither_skeleton_wall_skull", "minecraft:zombie_wall_head",
		"minecraft:player_wall_head", "minecraft:creeper_wall_head", "minecraft:dragon_wall_head", "minecraft:piglin_wall_head":
		return true
	default:
		return false
	}
}

func wallHeadOrSkullName(name string) string {
	switch name {
	case "skeleton_skull":
		return "skeleton_wall_skull"
	case "wither_skeleton_skull":
		return "wither_skeleton_wall_skull"
	case "zombie_head":
		return "zombie_wall_head"
	case "player_head":
		return "player_wall_head"
	case "creeper_head":
		return "creeper_wall_head"
	case "dragon_head":
		return "dragon_wall_head"
	case "piglin_head":
		return "piglin_wall_head"
	default:
		return name
	}
}

func attachmentBoolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func itoaSmall(value int) string {
	if value < 0 {
		value = 0
	}
	if value < 10 {
		return string(rune('0' + value))
	}
	return "1" + string(rune('0'+value-10))
}
