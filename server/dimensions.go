package server

import (
	"math"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

const (
	dimensionOverworld int32 = iota
	dimensionNether
	dimensionEnd
)

func (s *Server) tryBedrockPortalTravel(p *player.Player) bool {
	if p == nil || p.Edition != player.ClientEditionBedrock || time.Now().Before(p.PortalCooldownUntil) {
		return false
	}
	currentWorld := s.worldForPlayer(p)
	x := int(math.Floor(p.Position.X))
	y := int(math.Floor(p.Position.Y))
	z := int(math.Floor(p.Position.Z))
	portal := currentWorld.GetBlock(x, y, z).ResourceLocation()
	if portal == "minecraft:air" {
		portal = currentWorld.GetBlock(x, y+1, z).ResourceLocation()
	}

	var destination int32
	var target spatial.Vec3
	switch portal {
	case "minecraft:nether_portal":
		if p.Dimension == dimensionNether {
			destination = dimensionOverworld
			target = spatial.Vec3{X: p.Position.X * 8, Z: p.Position.Z * 8}
		} else if p.Dimension == dimensionOverworld {
			destination = dimensionNether
			target = spatial.Vec3{X: p.Position.X / 8, Z: p.Position.Z / 8}
		} else {
			return false
		}
	case "minecraft:end_portal":
		if p.Dimension == dimensionEnd {
			destination = dimensionOverworld
			target = p.WorldSpawn
		} else if p.Dimension == dimensionOverworld {
			destination = dimensionEnd
			target = spatial.Vec3{X: 100.5, Y: 50, Z: 0.5}
		} else {
			return false
		}
	default:
		return false
	}

	destinationWorld := s.worldForDimension(destination)
	if portal == "minecraft:nether_portal" {
		target.Y = float64(s.safePortalY(destinationWorld, int(math.Floor(target.X)), int(math.Floor(target.Z))))
		previous := s.bedrockActionWorld
		s.bedrockActionWorld = destinationWorld
		s.ensureNetherPortalAt(int(math.Floor(target.X)), int(target.Y)-1, int(math.Floor(target.Z)), "x")
		s.bedrockActionWorld = previous
		target.X = math.Floor(target.X) + 1.5
		target.Z = math.Floor(target.Z) + 0.5
	}
	p.Dimension = destination
	p.Position = target
	p.FallDistance = 0
	p.OnGround = false
	p.PortalCooldownUntil = time.Now().Add(4 * time.Second)
	if s.bedrockListener != nil {
		s.bedrockListener.ChangeDimension(p, destination, target)
	}
	return true
}

func (s *Server) safePortalY(world *coreworld.World, x, z int) int {
	for y := 32; y <= 118; y++ {
		if !world.GetBlock(x, y-1, z).IsAir() && world.GetBlock(x, y, z).IsAir() && world.GetBlock(x, y+1, z).IsAir() {
			return y
		}
	}
	return 64
}

// igniteNetherPortal recognises the standard 4x5 obsidian frame in either
// horizontal axis and fills its 2x3 interior.
func (s *Server) igniteNetherPortal(clickedX, clickedY, clickedZ int) bool {
	for _, axis := range []string{"x", "z"} {
		for horizontalOffset := -3; horizontalOffset <= 0; horizontalOffset++ {
			for verticalOffset := -4; verticalOffset <= 0; verticalOffset++ {
				left, bottom := horizontalOffset, clickedY+verticalOffset
				baseX, baseZ := clickedX, clickedZ
				if axis == "x" {
					baseX = clickedX + left
				} else {
					baseZ = clickedZ + left
				}
				if s.isNetherPortalFrame(baseX, bottom, baseZ, axis) {
					s.fillNetherPortal(baseX, bottom, baseZ, axis)
					return true
				}
			}
		}
	}
	return false
}

func (s *Server) isNetherPortalFrame(baseX, bottom, baseZ int, axis string) bool {
	world := s.bedrockWorld()
	for horizontal := 0; horizontal < 4; horizontal++ {
		for vertical := 0; vertical < 5; vertical++ {
			// Vanilla permits the four corner blocks to be omitted.
			border := (horizontal == 0 || horizontal == 3) && vertical >= 1 && vertical <= 3 ||
				(vertical == 0 || vertical == 4) && horizontal >= 1 && horizontal <= 2
			interior := horizontal >= 1 && horizontal <= 2 && vertical >= 1 && vertical <= 3
			x, z := baseX, baseZ
			if axis == "x" {
				x += horizontal
			} else {
				z += horizontal
			}
			block := world.GetBlock(x, bottom+vertical, z)
			if border && block.ResourceLocation() != "minecraft:obsidian" {
				return false
			}
			if interior && !bedrockPlacementReplaceable(block.ResourceLocation()) {
				return false
			}
		}
	}
	return true
}

func (s *Server) fillNetherPortal(baseX, bottom, baseZ int, axis string) {
	for horizontal := 1; horizontal <= 2; horizontal++ {
		for vertical := 1; vertical <= 3; vertical++ {
			x, z := baseX, baseZ
			if axis == "x" {
				x += horizontal
			} else {
				z += horizontal
			}
			s.setBedrockActionBlock(x, bottom+vertical, z, bedrockBlock("nether_portal", map[string]string{"axis": axis}))
		}
	}
}

func (s *Server) ensureNetherPortalAt(x, bottom, z int, axis string) {
	baseX, baseZ := x, z
	for horizontal := 0; horizontal < 4; horizontal++ {
		for vertical := 0; vertical < 5; vertical++ {
			border := horizontal == 0 || horizontal == 3 || vertical == 0 || vertical == 4
			px, pz := baseX, baseZ
			if axis == "x" {
				px += horizontal
			} else {
				pz += horizontal
			}
			if border {
				s.setBedrockActionBlock(px, bottom+vertical, pz, bedrockBlock("obsidian", nil))
			} else {
				s.setBedrockActionBlock(px, bottom+vertical, pz, bedrockBlock("nether_portal", map[string]string{"axis": axis}))
			}
		}
	}
}

func (s *Server) tryActivateEndPortal(frameX, frameY, frameZ int) bool {
	world := s.bedrockWorld()
	for originX := frameX - 4; originX <= frameX; originX++ {
		for originZ := frameZ - 4; originZ <= frameZ; originZ++ {
			valid := true
			for offset := 1; offset <= 3 && valid; offset++ {
				for _, expected := range []struct {
					x, z   int
					facing string
				}{
					{originX + offset, originZ, "south"},
					{originX + offset, originZ + 4, "north"},
					{originX, originZ + offset, "east"},
					{originX + 4, originZ + offset, "west"},
				} {
					frame := world.GetBlock(expected.x, frameY, expected.z)
					if frame.ResourceLocation() != "minecraft:end_portal_frame" || frame.Properties["eye"] != "true" ||
						frame.Properties["facing"] != expected.facing {
						valid = false
						break
					}
				}
			}
			if !valid {
				continue
			}
			for dx := 1; dx <= 3; dx++ {
				for dz := 1; dz <= 3; dz++ {
					s.setBedrockActionBlock(originX+dx, frameY, originZ+dz, bedrockBlock("end_portal", nil))
				}
			}
			return true
		}
	}
	return false
}
