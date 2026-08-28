package player

import (
	"testing"
	"time"
)

func TestPlayerActivityClock(t *testing.T) {
	p := New([16]byte{}, "Alex", ClientEditionJava)
	now := time.Now()
	if idle := p.IdleFor(now); idle < 0 || idle > time.Second {
		t.Fatalf("new player idle time = %v", idle)
	}
	p.activityUnix.Store(now.Add(-5 * time.Minute).UnixNano())
	if idle := p.IdleFor(now); idle != 5*time.Minute {
		t.Fatalf("idle time = %v", idle)
	}
}
