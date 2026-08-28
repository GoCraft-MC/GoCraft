package server

func (s *Server) currentDifficulty() int32 {
	if stored := s.difficulty.Load(); stored != 0 {
		return stored - 1
	}
	if s.cfg != nil {
		return difficultyID(s.cfg.Difficulty)
	}
	return 2
}
