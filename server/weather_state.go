package server

func (s *Server) currentWeather() (raining, thundering bool) {
	state := s.weather.Load()
	return state >= 1, state >= 2
}
