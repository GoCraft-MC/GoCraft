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
