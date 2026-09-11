package handler

import (
	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/session"
)

type placementCheck func(int, int, int, coreworld.Block) bool

func approvePlacement(checks []placementCheck, x, y, z int, placed coreworld.Block) bool {
	for _, check := range checks {
		if check != nil && !check(x, y, z, placed) {
			return false
		}
	}
	return true
}

// A multi-block placement publishes one event for its primary block, before
// either half, container contents, inventory or linked plants are changed.
func javaPlacementCheck(p *player.Player, w *coreworld.World, mgr *session.Manager, conn *network.ClientConn, seq int32, buses ...*coreplugin.Bus) placementCheck {
	return func(x, y, z int, placed coreworld.Block) bool {
		replaced := w.GetBlock(x, y, z)
		if len(buses) > 0 && buses[0] != nil && !buses[0].EmitBlockPlace(p,
			spatial.BlockPos{X: int32(x), Y: int32(y), Z: int32(z)}, placed, replaced, int64(p.Dimension)) {
			resyncPlacement(p, w, mgr, conn, x, y, z)
			sendAcknowledgeBlockChange(mgr, p, seq)
			return false
		}
		if !replaced.IsAir() && replaced.ResourceLocation() != "minecraft:water" && replaced.ResourceLocation() != "minecraft:lava" {
			breakLinkedPlantHalf(x, y, z, replaced, w, mgr)
		}
		broadcastSoundAt(mgr, blockBreakSound(placed.ResourceLocation()), soundCategoryBlocks,
			float64(x)+0.5, float64(y)+0.5, float64(z)+0.5, 1, 0.8)
		return true
	}
}

func resyncPlacement(p *player.Player, w *coreworld.World, mgr *session.Manager, conn *network.ClientConn, x, y, z int) {
	// Include neighbours: the client may predict a door/bed/chest partner.
	for _, offset := range [][3]int{{}, {0, 1, 0}, {0, -1, 0}, {1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}} {
		bx, by, bz := x+offset[0], y+offset[1], z+offset[2]
		if mgr != nil {
			BroadcastBlockChange(coreworld.BlockChange{X: bx, Y: by, Z: bz, Block: w.GetBlock(bx, by, bz)}, mgr)
		}
	}
	if conn != nil {
		_ = SyncPlayerInventory(conn, p)
	}
}
