package server

import (
	"fmt"
	"math"

	"GoCraft/core/spatial"
	"GoCraft/java/handler"
)

func (s *Server) registerVanillaSpawnCommands() {
	s.cmds.RegisterOperator("spawnpoint", s.commandSpawnPoint)
	s.cmds.RegisterOperator("setworldspawn", s.commandSetWorldSpawn)
}

func (s *Server) commandSpawnPoint(ctx handler.CommandContext) error {
	if ctx.Player == nil {
		return fmt.Errorf("player state is unavailable")
	}
	target, coordinateStart := ctx.Player, 0
	if len(ctx.Args) == 1 || len(ctx.Args) == 4 {
		coordinateStart = 1
		if ctx.Args[0] != "@s" {
			target = nil
			if ctx.FindPlayer != nil {
				target = ctx.FindPlayer(ctx.Args[0])
			}
			if target == nil {
				return fmt.Errorf("player not found: %s", ctx.Args[0])
			}
		}
	}
	if len(ctx.Args) != coordinateStart && len(ctx.Args) != coordinateStart+3 {
		return fmt.Errorf("usage: /spawnpoint [player] [x y z]")
	}
	position := target.Position
	if len(ctx.Args) == coordinateStart+3 {
		origins := []float64{target.Position.X, target.Position.Y, target.Position.Z}
		values := make([]int, 3)
		for index := range values {
			parsed, err := handler.ParseCommandCoordinate(ctx.Args[coordinateStart+index], origins[index])
			if err != nil {
				return err
			}
			values[index] = parsed
		}
		position = spatial.Vec3{X: float64(values[0]), Y: float64(values[1]), Z: float64(values[2])}
	}
	target.SpawnPoint = spatial.BlockPos{X: int32(math.Floor(position.X)), Y: int32(math.Floor(position.Y)), Z: int32(math.Floor(position.Z))}
	target.HasSpawnPoint = true
	return commandReply(ctx, fmt.Sprintf("Set spawn point for %s to %d %d %d", target.Username,
		target.SpawnPoint.X, target.SpawnPoint.Y, target.SpawnPoint.Z))
}

func (s *Server) commandSetWorldSpawn(ctx handler.CommandContext) error {
	if ctx.Player == nil || ctx.Player.Dimension != dimensionOverworld {
		return fmt.Errorf("world spawn can only be set in the Overworld")
	}
	if len(ctx.Args) != 0 && len(ctx.Args) != 3 {
		return fmt.Errorf("usage: /setworldspawn [x y z]")
	}
	position := ctx.Player.Position
	if len(ctx.Args) == 3 {
		origins := []float64{position.X, position.Y, position.Z}
		for index := range origins {
			value, err := handler.ParseCommandCoordinate(ctx.Args[index], origins[index])
			if err != nil {
				return err
			}
			origins[index] = float64(value)
		}
		position = spatial.Vec3{X: origins[0], Y: origins[1], Z: origins[2]}
	}
	position.X, position.Z = math.Floor(position.X)+0.5, math.Floor(position.Z)+0.5
	position.Y = math.Floor(position.Y)
	if err := s.setWorldSpawn(position); err != nil {
		return err
	}
	return commandReply(ctx, fmt.Sprintf("World spawn set to %.0f %.0f %.0f", position.X, position.Y, position.Z))
}
