package server

import (
	"fmt"

	"GoCraft/java/handler"
)

func (s *Server) registerReloadCommand() {
	s.cmds.RegisterOperator("reload", func(ctx handler.CommandContext) error {
		if len(ctx.Args) != 0 {
			return fmt.Errorf("usage: /reload")
		}
		if err := s.permissions.Reload(); err != nil {
			return fmt.Errorf("reloading permissions: %w", err)
		}
		if err := handler.ConfigureWhitelist("whitelist.json", s.cfg.Whitelist.Enabled, s.cfg.Whitelist.Players); err != nil {
			return fmt.Errorf("reloading whitelist: %w", err)
		}
		if err := handler.ConfigureBans("banned-players.json", "banned-ips.json"); err != nil {
			return fmt.Errorf("reloading bans: %w", err)
		}
		for _, online := range s.sessions.SnapshotAll() {
			_ = handler.SyncCommandPermissions(online.Conn, online.Player, s.cmds)
		}
		return commandReply(ctx, "Reloaded permissions, whitelist, and bans")
	})
}
