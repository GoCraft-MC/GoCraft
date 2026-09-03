package server

import (
	"fmt"
	"math"
	"sync"

	"GoCraft/config"
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

func (s *Server) setWorldSpawn(position spatial.Vec3) error {
	if s.cfg != nil && s.cfg.WorldStorage == config.WorldStorageDisk {
		if err := saveWorldSpawn(s.cfg.WorldDir, position); err != nil {
			return fmt.Errorf("saving world spawn: %w", err)
		}
	}
	if s.spawnState == nil {
		s.spawnState = newWorldSpawnState(position)
	} else {
		s.spawnState.set(position)
	}
	s.game.OnlinePlayers(func(online *player.Player) { online.WorldSpawn = position })
	if s.sessions != nil {
		for _, current := range s.sessions.SnapshotAll() {
			_ = handler.SyncDefaultSpawnPosition(current.Conn, current.Player)
		}
	}
	if s.bedrockListener != nil {
		s.bedrockListener.SetWorldSpawn(position)
	}
	return nil
}

func (s *Server) registerSpawnCommands() {
	s.registerVanillaSpawnCommands()
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
		if err := s.setWorldSpawn(position); err != nil {
			return err
		}
		return commandReply(ctx, fmt.Sprintf("World spawn set to %.1f %.0f %.1f", position.X, position.Y, position.Z))
	})
}

// commandReply answers the issuing player. The dispatcher fills Reply from an
// edition-neutral bridge before any handler runs, so a nil one means the
// command was invoked outside Dispatch and there is nobody to answer.
func commandReply(ctx handler.CommandContext, message string) error {
	if ctx.Reply == nil {
		return nil
	}
	return ctx.Reply(message)
}
