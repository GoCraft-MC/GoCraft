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
