package handler

import (
	"GoCraft/core/player"
	"GoCraft/java/network"
)

func resyncEventInventory(p *player.Player, conn *network.ClientConn, windowID int32) error {
	// A cancelled drag cannot carry a partial quick-craft selection forward.
	p.QuickCraftSlots = nil
	p.QuickCraftButton = 0
	p.ContainerStateID++
	if conn == nil {
		return nil
	}
	if windowID == 0 {
		return sendSetContainerContent(conn, p, p.ContainerStateID)
	}
	if p.OpenContainerKind == "minecraft:crafting_table" {
		return sendCraftingContainerContent(conn, p)
	}
	if IsFurnaceContainer(p.OpenContainerKind) {
		return sendFurnaceContainerContent(conn, p)
	}
	return sendChestContainerContent(conn, p)
}
