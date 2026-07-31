package server

import "fmt"

// PingInfo holds latency data for a connected player.
type PingInfo struct {
	PlayerName string
	LatencyMs  int
}

// String formats ping info for display in /list output.
func (p PingInfo) String() string {
	return fmt.Sprintf("%s (%dms)", p.PlayerName, p.LatencyMs)
}

// PingTracker keeps a rolling map of player pings updated each tick.
type PingTracker struct {
	pings map[string]int
}

// NewPingTracker creates an empty tracker.
func NewPingTracker() *PingTracker {
	return &PingTracker{pings: make(map[string]int)}
}

// Update records the latest latency for a player.
func (pt *PingTracker) Update(name string, ms int) {
	pt.pings[name] = ms
}

// Get returns the last recorded latency for a player, or -1 if unknown.
func (pt *PingTracker) Get(name string) int {
	if ms, ok := pt.pings[name]; ok {
		return ms
	}
	return -1
}
