package player

// ExperienceForLevel returns the total points at the beginning of a level.
func ExperienceForLevel(level int32) int32 {
	if level <= 0 {
		return 0
	}
	l := int64(level)
	var total int64
	switch {
	case level <= 16:
		total = l*l + 6*l
	case level <= 31:
		total = (5*l*l-81*l)/2 + 360
	default:
		total = (9*l*l-325*l)/2 + 2220
	}
	if total > 2147483647 {
		return 2147483647
	}
	return int32(total)
}

func ExperienceToNextLevel(level int32) int32 {
	switch {
	case level >= 30:
		return 9*level - 158
	case level >= 15:
		return 5*level - 38
	default:
		return 2*level + 7
	}
}

func levelForExperience(total int32) int32 {
	low, high := int32(0), int32(21864)
	for low+1 < high {
		middle := low + (high-low)/2
		if ExperienceForLevel(middle) <= total {
			low = middle
		} else {
			high = middle
		}
	}
	return low
}

func (p *Player) SetTotalExperience(total int32) {
	if total < 0 {
		total = 0
	}
	p.experienceMu.Lock()
	p.ExperienceTotal = total
	p.ExperienceLevel = levelForExperience(total)
	within := total - ExperienceForLevel(p.ExperienceLevel)
	p.ExperienceProgress = float32(within) / float32(ExperienceToNextLevel(p.ExperienceLevel))
	p.experienceMu.Unlock()
}

func (p *Player) SetExperienceLevel(level int32) {
	if level < 0 {
		level = 0
	}
	p.SetTotalExperience(ExperienceForLevel(level))
}

func (p *Player) AddExperience(points int32) {
	_, total, _ := p.ExperienceSnapshot()
	updated := int64(total) + int64(points)
	if updated > 2147483647 {
		updated = 2147483647
	} else if updated < 0 {
		updated = 0
	}
	p.SetTotalExperience(int32(updated))
}

func (p *Player) ExperienceSnapshot() (level, total int32, progress float32) {
	p.experienceMu.Lock()
	level, total, progress = p.ExperienceLevel, p.ExperienceTotal, p.ExperienceProgress
	p.experienceMu.Unlock()
	return
}
