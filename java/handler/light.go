package handler

import (
	"math"

	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

// BroadcastBlockLightUpdatesInDimension rebuilds the at-most-nine chunks whose
// block light may be affected by one canonical mutation. The update is scoped
// to Java viewers of the same dimension; Bedrock derives light client-side.
func BroadcastBlockLightUpdatesInDimension(world *coreworld.World, change coreworld.BlockChange, mgr *session.Manager, dimension int32) {
	if world == nil || mgr == nil || !javaworld.BlockLightChanged(change.Previous, change.Block) {
		return
	}
	viewers := make([]*session.Session, 0)
	for _, current := range mgr.SnapshotAll() {
		if current != nil && current.Conn != nil && current.Player != nil && current.Player.Dimension == dimension {
			viewers = append(viewers, current)
		}
	}
	if len(viewers) == 0 {
		return
	}
	cx := int32(math.Floor(float64(change.X) / coreworld.SectionSize))
	cz := int32(math.Floor(float64(change.Z) / coreworld.SectionSize))
	for dz := int32(-1); dz <= 1; dz++ {
		for dx := int32(-1); dx <= 1; dx++ {
			chunk, loaded := world.ChunkIfLoaded(cx+dx, cz+dz)
			if !loaded {
				continue
			}
			packet := javaworld.BuildBlockLightUpdate(world, chunk)
			if packet == nil {
				continue
			}
			for _, viewer := range viewers {
				_ = viewer.Conn.WritePacket(packet)
			}
		}
	}
}
