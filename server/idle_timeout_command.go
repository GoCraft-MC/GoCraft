package server

import (
	"fmt"
	"strconv"

	"GoCraft/java/handler"
)

func (s *Server) registerIdleTimeoutCommand() {
	s.cmds.RegisterOperator("setidletimeout", func(ctx handler.CommandContext) error {
		if len(ctx.Args) != 1 {
			return fmt.Errorf("usage: /setidletimeout <minutes>")
		}
		minutes, err := strconv.ParseInt(ctx.Args[0], 10, 32)
		if err != nil || minutes < 0 {
			return fmt.Errorf("idle timeout must be a non-negative number of minutes")
		}
		s.idleTimeout.Store(minutes)
		if minutes == 0 {
			return commandReply(ctx, "The player idle timeout is disabled")
		}
		return commandReply(ctx, fmt.Sprintf("The player idle timeout is now %d minutes", minutes))
	})
}
