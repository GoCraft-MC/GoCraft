package server

import "time"

// CreeperFuse tracks the fuse timer for a creeper entity.
type CreeperFuse struct {
	EntityID  int32
	Ignited   bool
	StartedAt time.Time
	Duration  time.Duration
}

// Tick advances the fuse and returns true when it should detonate.
func (f *CreeperFuse) Tick() bool {
	if !f.Ignited {
		return false
	}
	return time.Since(f.StartedAt) >= f.Duration
}

// Ignite starts the creeper fuse countdown.
func (f *CreeperFuse) Ignite() {
	f.Ignited = true
	f.StartedAt = time.Now()
	f.Duration = 1500 * time.Millisecond
}
