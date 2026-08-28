package server

import (
	"fmt"
	"strconv"
	"strings"

	"GoCraft/core/player"
	"GoCraft/java/handler"
)

func (s *Server) commandWeather(ctx handler.CommandContext) error {
	if len(ctx.Args) < 1 || len(ctx.Args) > 2 {
		return fmt.Errorf("usage: /weather <clear|rain|thunder> [duration-seconds]")
	}
	state := int32(-1)
	switch strings.ToLower(ctx.Args[0]) {
	case "clear":
		state = 0
	case "rain":
		state = 1
	case "thunder":
		state = 2
	}
	if state < 0 {
		return fmt.Errorf("unknown weather: %s", ctx.Args[0])
	}
	duration := int64(6000)
	if len(ctx.Args) == 2 {
		seconds, err := strconv.ParseInt(ctx.Args[1], 10, 32)
		if err != nil || seconds < 0 || seconds > 1_000_000 {
			return fmt.Errorf("duration must be 0..1000000 seconds")
		}
		duration = seconds * 20
	}
	s.setWeather(state, duration)
	return commandReply(ctx, "Weather set to "+strings.ToLower(ctx.Args[0]))
}

func (s *Server) setWeather(state int32, duration int64) {
	s.weather.Store(state)
	s.weatherTicks.Store(duration)
	raining, thundering := state >= 1, state >= 2
	if s.game != nil {
		s.game.OnlinePlayers(func(online *player.Player) {
			online.Raining, online.Thundering = raining, thundering
		})
	}
	event := byte(2)
	if raining {
		event = 1
	}
	handler.BroadcastGameEvent(s.sessions, event, 0)
	handler.BroadcastGameEvent(s.sessions, 7, boolLevel(raining))
	handler.BroadcastGameEvent(s.sessions, 8, boolLevel(thundering))
	if s.bedrockListener != nil {
		s.bedrockListener.SetWeather(raining, thundering)
	}
}

func boolLevel(enabled bool) float32 {
	if enabled {
		return 1
	}
	return 0
}

func (s *Server) tickWeather() {
	if remaining := s.weatherTicks.Load(); remaining > 0 && s.weatherTicks.Add(-1) == 0 {
		s.setWeather(0, 0)
	}
}
