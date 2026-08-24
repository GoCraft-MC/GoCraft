package server

import (
	"math"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func isMinecartRail(name string) bool {
	return name == "minecraft:rail" || name == "minecraft:powered_rail" ||
		name == "minecraft:detector_rail" || name == "minecraft:activator_rail"
}

func (s *Server) minecartRailAt(e *corentity.Entity) (coreworld.Block, int, int, int, bool) {
	x, y, z := int(math.Floor(e.Position.X)), int(math.Floor(e.Position.Y)), int(math.Floor(e.Position.Z))
	block := s.world.GetBlock(x, y, z)
	if isMinecartRail(block.ResourceLocation()) {
		return block, x, y, z, true
	}
	block = s.world.GetBlock(x, y-1, z)
	return block, x, y - 1, z, isMinecartRail(block.ResourceLocation())
}

func railTangent(shape string, vx, vz, yaw float64) (float64, float64) {
	axisX, axisZ := 0.0, 1.0
	switch shape {
	case "east_west", "ascending_east", "ascending_west":
		axisX, axisZ = 1, 0
	case "south_east":
		axisX, axisZ = 1, 1
	case "south_west":
		axisX, axisZ = -1, 1
	case "north_west":
		axisX, axisZ = -1, -1
	case "north_east":
		axisX, axisZ = 1, -1
	}
	length := math.Hypot(axisX, axisZ)
	axisX, axisZ = axisX/length, axisZ/length
	if vx*axisX+vz*axisZ < 0 || (math.Hypot(vx, vz) < 0.001 && (-math.Sin(yaw))*axisX+math.Cos(yaw)*axisZ < 0) {
		axisX, axisZ = -axisX, -axisZ
	}
	return axisX, axisZ
}

func (s *Server) tickMinecartPhysics(e *corentity.Entity) {
	rail, railX, railY, railZ, onRail := s.minecartRailAt(e)
	s.tickMinecartRailEffects(e, rail, railX, railY, railZ, onRail)
	if !onRail {
		e.VY -= 0.04
		e.Position.X += e.VX
		e.Position.Y += e.VY
		e.Position.Z += e.VZ
		e.VX *= 0.98
		e.VZ *= 0.98
		return
	}
	shape := rail.Properties["shape"]
	tx, tz := railTangent(shape, e.VX, e.VZ, float64(e.Yaw)*math.Pi/180)
	speed := e.VX*tx + e.VZ*tz
	if speed < 0 {
		speed = -speed
		tx, tz = -tx, -tz
	}
	if rail.ResourceLocation() == "minecraft:powered_rail" {
		if rail.Properties["powered"] == "true" {
			if speed < 0.01 {
				speed = 0.1
			} else {
				speed = math.Min(0.4, speed+0.06)
			}
		} else {
			speed *= 0.5
			if speed < 0.01 {
				speed = 0
			}
		}
	}
	if speed > 0.4 {
		speed = 0.4
	}
	e.VX, e.VZ, e.VY = tx*speed, tz*speed, 0
	e.Position.X += e.VX
	e.Position.Z += e.VZ
	e.Position.Y = float64(railY) + 0.0625
	if tx == 0 {
		e.Position.X = float64(railX) + 0.5
	} else if tz == 0 {
		e.Position.Z = float64(railZ) + 0.5
	}
	if shape == "ascending_east" || shape == "ascending_west" || shape == "ascending_north" || shape == "ascending_south" {
		e.Position.Y += 0.5
	}
	e.Yaw = float32(math.Atan2(-e.VX, e.VZ) * 180 / math.Pi)
	friction := 0.96
	if e.RiderEntityID != 0 {
		friction = 0.99
	}
	e.VX, e.VZ = e.VX*friction, e.VZ*friction
}

func (s *Server) tickMinecartRailEffects(e *corentity.Entity, rail coreworld.Block, x, y, z int, onRail bool) {
	isDetector := onRail && rail.ResourceLocation() == "minecraft:detector_rail"
	if e.MinecartOnDetector && (!isDetector || x != e.MinecartDetectorX || y != e.MinecartDetectorY || z != e.MinecartDetectorZ) {
		previous := s.world.GetBlock(e.MinecartDetectorX, e.MinecartDetectorY, e.MinecartDetectorZ)
		if previous.ResourceLocation() == "minecraft:detector_rail" {
			previous.Properties = copyStringMap(previous.Properties)
			previous.Properties["powered"] = "false"
			s.world.SetBlock(e.MinecartDetectorX, e.MinecartDetectorY, e.MinecartDetectorZ, previous)
		}
		e.MinecartOnDetector = false
	}
	if isDetector {
		if rail.Properties["powered"] != "true" {
			rail.Properties = copyStringMap(rail.Properties)
			rail.Properties["powered"] = "true"
			s.world.SetBlock(x, y, z, rail)
		}
		e.MinecartDetectorX, e.MinecartDetectorY, e.MinecartDetectorZ = x, y, z
		e.MinecartOnDetector = true
	}
	if onRail && rail.ResourceLocation() == "minecraft:activator_rail" && rail.Properties["powered"] == "true" {
		s.dismountEntityPassengers(e)
		if e.Type == corentity.TypeTNTMinecart && e.FuseTicks < 0 {
			e.FuseTicks = 80
		}
	}
}

func copyStringMap(source map[string]string) map[string]string {
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func (s *Server) tickTNTMinecartFuse(e *corentity.Entity) bool {
	if e.Type != corentity.TypeTNTMinecart || e.FuseTicks <= 0 {
		return false
	}
	e.FuseTicks--
	if e.FuseTicks != 0 {
		return false
	}
	s.explodeTNT(e.Position.X, e.Position.Y, e.Position.Z)
	return true
}

func tickMinecartCollisions(cart *corentity.Entity, entities []*corentity.Entity) {
	for _, other := range entities {
		if other == nil || other == cart || other.Dead || other.EntityID < cart.EntityID ||
			other.EntityID == cart.RiderEntityID {
			continue
		}
		dx, dz := other.Position.X-cart.Position.X, other.Position.Z-cart.Position.Z
		distanceSquared := dx*dx + dz*dz
		if distanceSquared < 1.0e-4 || distanceSquared >= 1 {
			continue
		}
		distance := math.Sqrt(distanceSquared)
		forceX, forceZ := dx/distance*0.05, dz/distance*0.05
		if corentity.IsMinecart(other.Type) {
			averageX, averageZ := (cart.VX+other.VX)/2, (cart.VZ+other.VZ)/2
			if cart.Type == corentity.TypeFurnaceMinecart && other.Type != corentity.TypeFurnaceMinecart {
				other.VX, other.VZ = other.VX*0.2+cart.VX-forceX, other.VZ*0.2+cart.VZ-forceZ
				cart.VX, cart.VZ = cart.VX*0.95, cart.VZ*0.95
			} else if other.Type == corentity.TypeFurnaceMinecart && cart.Type != corentity.TypeFurnaceMinecart {
				cart.VX, cart.VZ = cart.VX*0.2+other.VX+forceX, cart.VZ*0.2+other.VZ+forceZ
				other.VX, other.VZ = other.VX*0.95, other.VZ*0.95
			} else {
				cart.VX, cart.VZ = cart.VX*0.2+averageX-forceX, cart.VZ*0.2+averageZ-forceZ
				other.VX, other.VZ = other.VX*0.2+averageX+forceX, other.VZ*0.2+averageZ+forceZ
			}
			continue
		}
		cart.VX, cart.VZ = cart.VX-forceX, cart.VZ-forceZ
		other.VX, other.VZ = other.VX+forceX, other.VZ+forceZ
	}
}
