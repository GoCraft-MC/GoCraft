package server

import (
	"fmt"

	"GoCraft/core/player"
	"GoCraft/java/handler"
)

func (s *Server) disconnectPlayer(target *player.Player, reason string) error {
	if target == nil {
		return fmt.Errorf("target player is unavailable")
	}
	if target.Edition == player.ClientEditionJava && handler.DisconnectJavaPlayer(target, s.sessions, reason) {
		return nil
	}
	if target.Edition == player.ClientEditionBedrock && s.bedrockListener != nil && s.bedrockListener.DisconnectPlayer(target.UUID, reason) {
		return nil
	}
	return fmt.Errorf("player session is unavailable")
}
