package handler

import (
	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/session"
)

func javaMinecartType(item string) (corentity.EntityType, bool) {
	switch item {
	case "minecraft:minecart":
		return corentity.TypeMinecart, true
	case "minecraft:chest_minecart":
		return corentity.TypeChestMinecart, true
	case "minecraft:furnace_minecart":
		return corentity.TypeFurnaceMinecart, true
	case "minecraft:hopper_minecart":
		return corentity.TypeHopperMinecart, true
	case "minecraft:tnt_minecart":
		return corentity.TypeTNTMinecart, true
	case "minecraft:command_block_minecart":
		return corentity.TypeCommandMinecart, true
	default:
		return "", false
	}
}

func placeJavaMinecart(p *player.Player, conn *network.ClientConn, mgr *session.Manager, w *coreworld.World,
	x, y, z int, nextEntityID func() int32) bool {
	entityType, minecart := javaMinecartType(p.HeldItem().ItemID)
	if !minecart || nextEntityID == nil {
		return false
	}
	rail := w.GetBlock(x, y, z)
	if !isJavaMinecartRail(rail.ResourceLocation()) {
		return true
	}
	entity := corentity.New(nextEntityID(), [16]byte{}, entityType, float64(x)+0.5, float64(y)+0.0625, float64(z)+0.5)
	entity.Yaw = p.Rotation.Yaw
	w.Entities.Add(entity)
	BroadcastSpawnMob(entity, mgr)
	if p.GameMode == player.GameModeSurvival {
		slot := player.HotbarStart + p.HeldSlot
		p.Inventory[slot].Count--
		normalizeStack(&p.Inventory[slot])
		p.ContainerStateID++
		if conn != nil {
			_ = SyncPlayerInventory(conn, p)
		}
	}
	return true
}

func isJavaMinecartRail(name string) bool {
	return name == "minecraft:rail" || name == "minecraft:powered_rail" ||
		name == "minecraft:detector_rail" || name == "minecraft:activator_rail"
}
