package server

import (
	"fmt"
	"math"
	"sync"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/handler"
)

type worldSpawnState struct {
	mu       sync.RWMutex
	position spatial.Vec3
}

func newWorldSpawnState(position spatial.Vec3) *worldSpawnState {
	return &worldSpawnState{position: position}
}
func (state *worldSpawnState) get() spatial.Vec3 {
	state.mu.RLock()
	position := state.position
	state.mu.RUnlock()
	return position
}
func (state *worldSpawnState) set(position spatial.Vec3) {
	state.mu.Lock()
	state.position = position
	state.mu.Unlock()
}

func (s *Server) currentWorldSpawn() spatial.Vec3 {
	if s.spawnState != nil {
		return s.spawnState.get()
	}
	return spatial.Vec3{X: float64(s.spawnX) + 0.5, Y: float64(s.safeSpawnY(s.spawnX, s.spawnZ)), Z: float64(s.spawnZ) + 0.5}
}

func (s *Server) setWorldSpawn(position spatial.Vec3) {
	if s.spawnState == nil {
		s.spawnState = newWorldSpawnState(position)
	} else {
		s.spawnState.set(position)
	}
	s.game.OnlinePlayers(func(online *player.Player) { online.WorldSpawn = position })
}

func (s *Server) registerSpawnCommands() {
	s.cmds.Register("spawn", func(ctx handler.CommandContext) error {
		if ctx.Player == nil || ctx.TeleportTo == nil {
			return fmt.Errorf("teleport service is unavailable")
		}
		if len(ctx.Args) != 0 {
			return fmt.Errorf("usage: /spawn")
		}
		position := s.currentWorldSpawn()
		ctx.Player.WorldSpawn = position
		if ctx.Player.Dimension != dimensionOverworld {
			if ctx.ChangeWorld == nil {
				return fmt.Errorf("dimension changing is unavailable")
			}
			if err := ctx.ChangeWorld(dimensionOverworld); err != nil {
				return fmt.Errorf("returning to Overworld: %w", err)
			}
		}
		if err := ctx.TeleportTo(position.X, position.Y, position.Z); err != nil {
			return fmt.Errorf("teleporting to spawn: %w", err)
		}
		return commandReply(ctx, "Teleported to spawn")
	})
	s.cmds.RegisterOperator("setspawn", func(ctx handler.CommandContext) error {
		if ctx.Player == nil || ctx.Player.Dimension != dimensionOverworld {
			return fmt.Errorf("world spawn can only be set in the Overworld")
		}
		if len(ctx.Args) != 0 {
			return fmt.Errorf("usage: /setspawn")
		}
		position := spatial.Vec3{X: math.Floor(ctx.Player.Position.X) + 0.5, Y: math.Floor(ctx.Player.Position.Y), Z: math.Floor(ctx.Player.Position.Z) + 0.5}
		s.setWorldSpawn(position)
		return commandReply(ctx, fmt.Sprintf("World spawn set to %.1f %.0f %.1f", position.X, position.Y, position.Z))
	})
}

func commandReply(ctx handler.CommandContext, message string) error {
	if ctx.Reply != nil {
		return ctx.Reply(message)
	}
	return handler.SendSystemMessage(ctx.Conn, message)
}
