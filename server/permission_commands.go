package server

import (
	"fmt"
	"strings"

	"GoCraft/java/handler"
)

func (s *Server) registerPermissionCommands() {
	s.cmds.RegisterOperator("gocraft", func(ctx handler.CommandContext) error {
		message, err := s.executePermissionCommand(ctx.Args)
		if err != nil {
			return err
		}
		return commandReply(ctx, message)
	})
}

func (s *Server) executePermissionCommand(arguments []string) (string, error) {
	if s.permissionEditor == nil {
		return "", fmt.Errorf("permission editor is disabled in server.yml")
	}
	if len(arguments) == 1 && strings.EqualFold(arguments[0], "peditor") {
		link, err := s.permissionEditor.create(s.cmds.CommandPermissions())
		if err != nil {
			return "", fmt.Errorf("creating permission editor: %w", err)
		}
		return "Open the permission editor: " + link, nil
	}
	if len(arguments) == 2 && strings.EqualFold(arguments[0], "applyedits") {
		if err := s.permissionEditor.apply(arguments[1]); err != nil {
			return "", fmt.Errorf("applying permission edits: %w", err)
		}
		return "Permission edits applied and saved to permissions.json", nil
	}
	return "", fmt.Errorf("usage: /gocraft <peditor|applyedits <link-or-code>>")
}
