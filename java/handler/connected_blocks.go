package handler

import (
	"strconv"
	"strings"

	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

// refreshJavaConnectedBlocks recalculates the explicit Java block states used
// by fences, panes, iron bars, walls and fence gates after a neighbour changes.
// Bedrock has a separate runtime-state conversion pass; Java cannot rely on a
// client-side model refresh because each connection is part of the state ID.
func refreshJavaConnectedBlocks(x, y, z int, w *coreworld.World, mgr *session.Manager) {
	if w == nil {
		return
	}
	positions := [][3]int{
		{x, y, z}, {x - 1, y, z}, {x + 1, y, z}, {x, y, z - 1}, {x, y, z + 1}, {x, y - 1, z},
	}
	for _, position := range positions {
		px, py, pz := position[0], position[1], position[2]
		if py < coreworld.WorldMinY || py > coreworld.WorldMaxY {
			continue
		}
		block := w.GetBlock(px, py, pz)
		var updated coreworld.Block
		switch name := block.ResourceLocation(); {
		case javaIsFence(name):
			updated = javaFenceState(px, py, pz, block, w)
		case javaIsPane(name):
			updated = javaPaneState(px, py, pz, block, w)
		case javaIsWall(name):
			updated = javaWallState(px, py, pz, block, w)
		case strings.HasSuffix(name, "_fence_gate"):
			updated = copyBlockProperties(block)
			updated.Properties["in_wall"] = strconv.FormatBool(javaGateInWall(px, py, pz, block, w))
		default:
			continue
		}
		if updated.Key() == block.Key() {
			continue
		}
		w.SetBlock(px, py, pz, updated)
		broadcastJavaConnectedBlock(px, py, pz, updated, mgr)
	}
}

func javaFenceState(x, y, z int, block coreworld.Block, w *coreworld.World) coreworld.Block {
	updated := copyBlockProperties(block)
	for _, direction := range javaHorizontalDirections {
		neighbor := w.GetBlock(x+direction.dx, y, z+direction.dz)
		updated.Properties[direction.name] = strconv.FormatBool(javaFenceConnectsTo(block.ResourceLocation(), neighbor, direction.name))
	}
	if _, ok := updated.Properties["waterlogged"]; !ok {
		updated.Properties["waterlogged"] = "false"
	}
	return updated
}

func javaPaneState(x, y, z int, block coreworld.Block, w *coreworld.World) coreworld.Block {
	updated := copyBlockProperties(block)
	for _, direction := range javaHorizontalDirections {
		neighbor := w.GetBlock(x+direction.dx, y, z+direction.dz)
		updated.Properties[direction.name] = strconv.FormatBool(javaIsPane(neighbor.ResourceLocation()) || javaConnectedFullFace(neighbor))
	}
	if _, ok := updated.Properties["waterlogged"]; !ok {
		updated.Properties["waterlogged"] = "false"
	}
	return updated
}

func javaWallState(x, y, z int, block coreworld.Block, w *coreworld.World) coreworld.Block {
	updated := copyBlockProperties(block)
	connections := 0
	tall := javaConnectedFullFace(w.GetBlock(x, y+1, z)) || javaIsWall(w.GetBlock(x, y+1, z).ResourceLocation())
	for _, direction := range javaHorizontalDirections {
		value := "none"
		neighbor := w.GetBlock(x+direction.dx, y, z+direction.dz)
		if javaWallConnectsTo(neighbor, direction.name) {
			value = "low"
			if tall {
				value = "tall"
			}
			connections++
		}
		updated.Properties[direction.name] = value
	}
	north, east := updated.Properties["north"] != "none", updated.Properties["east"] != "none"
	south, west := updated.Properties["south"] != "none", updated.Properties["west"] != "none"
	post := connections < 2
	if connections >= 2 {
		switch {
		case north && south:
			post = east || west
		case east && west:
			post = north || south
		default:
			post = true
		}
	}
	updated.Properties["up"] = strconv.FormatBool(post)
	if _, ok := updated.Properties["waterlogged"]; !ok {
		updated.Properties["waterlogged"] = "false"
	}
	return updated
}

var javaHorizontalDirections = [...]struct {
	name   string
	dx, dz int
}{
	{name: "north", dz: -1}, {name: "east", dx: 1},
	{name: "south", dz: 1}, {name: "west", dx: -1},
}

func javaFenceConnectsTo(fence string, neighbor coreworld.Block, direction string) bool {
	name := neighbor.ResourceLocation()
	if javaIsFence(name) {
		return (fence == "minecraft:nether_brick_fence") == (name == "minecraft:nether_brick_fence")
	}
	if strings.HasSuffix(name, "_fence_gate") {
		facing := neighbor.Properties["facing"]
		if direction == "north" || direction == "south" {
			return facing == "east" || facing == "west"
		}
		return facing == "north" || facing == "south"
	}
	return javaConnectedFullFace(neighbor)
}

func javaWallConnectsTo(block coreworld.Block, direction string) bool {
	name := block.ResourceLocation()
	if javaIsWall(name) || javaIsPane(name) {
		return true
	}
	if strings.HasSuffix(name, "_fence_gate") {
		facing := block.Properties["facing"]
		if direction == "north" || direction == "south" {
			return facing == "east" || facing == "west"
		}
		return facing == "north" || facing == "south"
	}
	return javaConnectedFullFace(block)
}

func javaConnectedFullFace(block coreworld.Block) bool {
	name := block.ResourceLocation()
	if placementReplaceable(name) || coreworld.IsFluidBlock(name) || javaIsFence(name) || javaIsPane(name) || javaIsWall(name) ||
		strings.HasSuffix(name, "_fence_gate") || strings.HasSuffix(name, "_door") || isTrapdoorBlock(name) ||
		strings.HasSuffix(name, "_button") || strings.HasSuffix(name, "_pressure_plate") || strings.Contains(name, "torch") ||
		strings.HasSuffix(name, "_sign") || strings.HasSuffix(name, "_wall_sign") || name == "minecraft:lever" ||
		name == "minecraft:ladder" || name == "minecraft:chain" || name == "minecraft:lantern" {
		return false
	}
	return name != ""
}

func javaGateInWall(x, y, z int, gate coreworld.Block, w *coreworld.World) bool {
	facing := gate.Properties["facing"]
	if facing == "north" || facing == "south" {
		return javaIsWall(w.GetBlock(x-1, y, z).ResourceLocation()) || javaIsWall(w.GetBlock(x+1, y, z).ResourceLocation())
	}
	return javaIsWall(w.GetBlock(x, y, z-1).ResourceLocation()) || javaIsWall(w.GetBlock(x, y, z+1).ResourceLocation())
}

func javaIsFence(name string) bool {
	return strings.HasSuffix(name, "_fence") && !strings.HasSuffix(name, "_fence_gate")
}

func javaIsPane(name string) bool {
	return name == "minecraft:iron_bars" || name == "minecraft:glass_pane" || strings.HasSuffix(name, "_stained_glass_pane")
}

func javaIsWall(name string) bool { return strings.HasSuffix(name, "_wall") }

func broadcastJavaConnectedBlock(x, y, z int, block coreworld.Block, mgr *session.Manager) {
	if mgr == nil {
		return
	}
	pkt := buildBlockUpdate(x, y, z, javaworld.StateID(block))
	for _, current := range mgr.SnapshotAll() {
		_ = current.Conn.WritePacket(pkt)
	}
}
