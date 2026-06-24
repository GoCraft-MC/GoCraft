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
	if p == nil || time.Now().Before(p.PortalCooldownUntil) {
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
	p.PortalCooldownUntil = time.Now().Add(4 * time.Second)
	target = destinationWorld.EnsureSafeArrival(target, destination)
	p.InvulnerableUntil = time.Now().Add(10 * time.Second)
	if p.Edition == player.ClientEditionJava {
		if javaSession, ok := s.sessions.Get(p.UUID); ok && javaSession.ChangeDimension != nil {
			if err := javaSession.ChangeDimension(destination, target); err == nil {
				return true
			}
		}
		return false
	}
	p.Dimension = destination
	p.Position = target
	p.FallDistance = 0
	p.OnGround = false
	if s.bedrockListener != nil {
		s.bedrockListener.ChangeDimension(p, destination, target)
	}
	return true
}

func (s *Server) commandWorldTarget(p *player.Player, destination int32) spatial.Vec3 {
	world := s.worldForDimension(destination)
	if destination == dimensionOverworld {
		return p.WorldSpawn
	}
	if destination == dimensionEnd {
		x, z := 100, 0
		return spatial.Vec3{X: 100.5, Y: float64(world.SurfaceY(x, z) + 1), Z: 0.5}
	}
	x := int(math.Floor(p.Position.X / 8))
	z := int(math.Floor(p.Position.Z / 8))
	y := s.safePortalY(world, x, z)
	return spatial.Vec3{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5}
}

// ensurePlayerPositionClear preserves a saved location when its feet and head
// are passable. If blocks were placed into that space while the player was
// offline, it moves them to the nearest supported two-block-high opening.
func (s *Server) ensurePlayerPositionClear(p *player.Player) bool {
	if p == nil {
		return false
	}
	w := s.worldForPlayer(p)
	x := int(math.Floor(p.Position.X))
	y := int(math.Floor(p.Position.Y))
	z := int(math.Floor(p.Position.Z))
	if playerStandingSpace(w, x, y, z) {
		return false
	}
	for distance := 1; distance <= 16; distance++ {
		for _, candidateY := range []int{y + distance, y - distance} {
			if candidateY < coreworld.WorldMinY+1 || candidateY >= coreworld.WorldMaxY || !playerStandingSpace(w, x, candidateY, z) {
				continue
			}
			p.Position = spatial.Vec3{X: p.Position.X, Y: float64(candidateY), Z: p.Position.Z}
			return true
		}
	}
	if p.Dimension == dimensionOverworld {
		p.Position = p.WorldSpawn
	} else {
		p.Position = w.EnsureSafeArrival(s.commandWorldTarget(p, p.Dimension), p.Dimension)
	}
	return true
}

func playerStandingSpace(w *coreworld.World, x, y, z int) bool {
	if w == nil || y <= coreworld.WorldMinY || y >= coreworld.WorldMaxY {
		return false
	}
	feet := w.GetBlock(x, y, z).ResourceLocation()
	head := w.GetBlock(x, y+1, z).ResourceLocation()
	below := w.GetBlock(x, y-1, z).ResourceLocation()
	return !coreworld.IsEntitySupportBlock(feet) && !coreworld.IsEntitySupportBlock(head) &&
		coreworld.IsEntitySupportBlock(below) && !coreworld.IsFluidBlock(below)
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
	changes, ok := coreworld.NetherPortalInterior(s.bedrockWorld(), clickedX, clickedY, clickedZ)
	if !ok {
		return false
	}
	for _, change := range changes {
		s.setBedrockActionBlock(change.X, change.Y, change.Z, change.Block)
	}
	return true
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
