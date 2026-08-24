package world

import "strings"

// ApplyPistonPower extends or retracts a piston using the canonical world.
// It moves at most twelve blocks, matching the vanilla/Pumpkin push limit.
func (w *World) ApplyPistonPower(x, y, z int, powered bool) []BlockChange {
	piston := w.GetBlock(x, y, z)
	name := piston.ResourceLocation()
	if name != "minecraft:piston" && name != "minecraft:sticky_piston" {
		return nil
	}
	dx, dy, dz := pistonOffset(piston.Properties["facing"])
	if powered == (piston.Properties["extended"] == "true") {
		return nil
	}
	changes := make([]BlockChange, 0, 15)
	set := func(px, py, pz int, block Block) {
		w.SetBlock(px, py, pz, block)
		changes = append(changes, BlockChange{X: px, Y: py, Z: pz, Block: block})
	}
	if powered {
		line := make([]Block, 0, 12)
		for distance := 1; distance <= 13; distance++ {
			block := w.GetBlock(x+dx*distance, y+dy*distance, z+dz*distance)
			if block.IsAir() || pistonReplaceable(block.ResourceLocation()) {
				break
			}
			if distance == 13 || pistonImmovable(block.ResourceLocation()) {
				return nil
			}
			line = append(line, block)
		}
		for index := len(line) - 1; index >= 0; index-- {
			distance := index + 2
			set(x+dx*distance, y+dy*distance, z+dz*distance, line[index])
		}
		head := Block{Namespace: "minecraft", Name: "piston_head", Properties: map[string]string{
			"facing": piston.Properties["facing"], "short": "false", "type": "normal",
		}}
		if name == "minecraft:sticky_piston" {
			head.Properties["type"] = "sticky"
		}
		set(x+dx, y+dy, z+dz, head)
	} else {
		frontX, frontY, frontZ := x+dx, y+dy, z+dz
		if w.GetBlock(frontX, frontY, frontZ).ResourceLocation() == "minecraft:piston_head" {
			set(frontX, frontY, frontZ, Air)
		}
		if name == "minecraft:sticky_piston" {
			pullX, pullY, pullZ := x+dx*2, y+dy*2, z+dz*2
			pulled := w.GetBlock(pullX, pullY, pullZ)
			if !pulled.IsAir() && !pistonImmovable(pulled.ResourceLocation()) {
				set(frontX, frontY, frontZ, pulled)
				set(pullX, pullY, pullZ, Air)
			}
		}
	}
	updated := redstoneBlockWith(piston, "extended", boolStr(powered))
	set(x, y, z, updated)
	return changes
}

func pistonOffset(facing string) (int, int, int) {
	switch facing {
	case "down":
		return 0, -1, 0
	case "up":
		return 0, 1, 0
	case "south":
		return 0, 0, 1
	case "east":
		return 1, 0, 0
	case "west":
		return -1, 0, 0
	default:
		return 0, 0, -1
	}
}

func pistonReplaceable(name string) bool {
	return name == "minecraft:water" || name == "minecraft:lava" || name == "minecraft:fire" ||
		strings.HasSuffix(name, "_grass") || strings.HasSuffix(name, "_flower")
}

func pistonImmovable(name string) bool {
	switch name {
	case "minecraft:bedrock", "minecraft:barrier", "minecraft:obsidian", "minecraft:crying_obsidian",
		"minecraft:reinforced_deepslate", "minecraft:end_portal", "minecraft:end_portal_frame",
		"minecraft:nether_portal", "minecraft:moving_piston", "minecraft:piston_head":
		return true
	}
	return false
}
