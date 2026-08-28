package server

import (
	"time"

	"GoCraft/core/player"
)

func (s *Server) tickIdleTimeout() {
	minutes := s.idleTimeout.Load()
	if minutes <= 0 || s.worldAge%20 != 0 || s.game == nil {
		return
	}
	now := time.Now()
	limit := time.Duration(minutes) * time.Minute
	var idle []*player.Player
	s.game.OnlinePlayers(func(online *player.Player) {
		if online.IdleFor(now) >= limit {
			idle = append(idle, online)
		}
	})
	for _, online := range idle {
		_ = s.disconnectPlayer(online, "You have been idle for too long!")
	}
}
