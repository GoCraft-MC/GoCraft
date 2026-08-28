package server

import (
	"errors"
	"fmt"

	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

func (s *Server) registerLifecycleCommands() {
	s.cmds.RegisterOperator("save-all", func(ctx handler.CommandContext) error {
		if err := s.saveWorldState(); err != nil {
			return fmt.Errorf("saving world: %w", err)
		}
		return commandReply(ctx, "Saved the game")
	})
	s.cmds.RegisterOperator("save-off", func(ctx handler.CommandContext) error {
		s.autosaveEnabled.Store(false)
		return commandReply(ctx, "Disabled automatic saving")
	})
	s.cmds.RegisterOperator("save-on", func(ctx handler.CommandContext) error {
		s.autosaveEnabled.Store(true)
		return commandReply(ctx, "Enabled automatic saving")
	})
	s.cmds.RegisterOperator("stop", func(ctx handler.CommandContext) error {
		if err := commandReply(ctx, "Stopping the server"); err != nil {
			return err
		}
		s.stopOnce.Do(func() { close(s.stopRequested) })
		return nil
	})
}

func (s *Server) saveWorldState() error {
	var saveErr error
	for _, dimensionWorld := range []*coreworld.World{s.world, s.netherWorld, s.endWorld} {
		if dimensionWorld != nil {
			saveErr = errors.Join(saveErr, dimensionWorld.Flush())
		}
	}
	s.saveWorldAge()
	s.saveAllPlayerData()
	return saveErr
}
