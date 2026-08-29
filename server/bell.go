package server

import (
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

func (s *Server) applyBellRing(i intent.BellRingIntent) {
	p := s.game.GetPlayer(i.PlayerUUID)
	if p == nil || p.Dead || p.GameMode == player.GameModeSpectator {
		return
	}
	world := s.worldForPlayer(p)
	if world == nil {
		return
	}
	centre := spatial.Vec3{X: float64(i.Position.X) + 0.5, Y: float64(i.Position.Y) + 0.5, Z: float64(i.Position.Z) + 0.5}
	if p.Position.Distance(centre) > 6.5 {
		return
	}
	block := world.GetBlock(int(i.Position.X), int(i.Position.Y), int(i.Position.Z))
	direction, valid := coreworld.BellRingDirection(block, i.Face, i.HitY)
	if valid {
		s.ringBell(world, p.Dimension, i.Position, direction)
	}
}

func (s *Server) ringBell(world *coreworld.World, dimension int32, position spatial.BlockPos, direction string) {
	if world == nil || world.GetBlock(int(position.X), int(position.Y), int(position.Z)).ResourceLocation() != "minecraft:bell" {
		return
	}
	handler.BroadcastBellRing(position, direction, dimension, s.sessions)
	if s.bedrockListener != nil {
		s.bedrockListener.BroadcastBellRing(dimension, position, direction)
	}
	world.EmitVibration(int(position.X), int(position.Y), int(position.Z))
}
