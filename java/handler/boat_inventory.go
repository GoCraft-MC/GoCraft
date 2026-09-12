package handler

import (
	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

const boatContainerKind = "minecraft:chest_boat"

func canAccessBoatInventory(p *player.Player, boat *corentity.Entity) bool {
	return p != nil && boat != nil && !p.Dead && !boat.Dead &&
		p.GameMode != player.GameModeSpectator && corentity.IsChestBoat(boat.Type) &&
		(p.Position.Distance(boat.Position) <= 4 || (p.VehicleEntityID == boat.EntityID && boat.HasPassenger(p.EntityID)))
}

func openBoatInventory(p *player.Player, conn *network.ClientConn, boat *corentity.Entity) error {
	if !canAccessBoatInventory(p, boat) || boat.Storage == nil {
		return nil
	}
	if p.OpenContainerKind == "minecraft:crafting_table" {
		returnCraftingGrid(p)
	}
	p.OpenContainerID = chestContainerID
	p.OpenContainerKind = boatContainerKind
	p.OpenContainerPos, p.OpenContainerPartnerPos = spatial.BlockPos{}, spatial.BlockPos{}
	p.OpenContainerHasPartner = false
	p.OpenContainerEntityID, p.OpenContainerStorage = boat.EntityID, boat.Storage
	p.ContainerSlots = boat.Storage.Snapshot()
	p.ContainerStateID++
	if conn == nil {
		return nil
	}
	if err := sendOpenScreen(conn, chestContainerID, containerMenuType("minecraft:chest"), "Boat Chest"); err != nil {
		return err
	}
	return sendChestContainerContent(conn, p)
}

func validBoatInventory(p *player.Player, w *coreworld.World) bool {
	if w == nil {
		return false
	}
	boat, ok := w.Entities.Get(p.OpenContainerEntityID)
	return ok && canAccessBoatInventory(p, boat) && boat.Storage == p.OpenContainerStorage
}

func closeBoatInventory(p *player.Player, conn *network.ClientConn) error {
	if p.OpenContainerKind != boatContainerKind {
		return nil
	}
	p.OpenContainerID, p.OpenContainerEntityID = 0, 0
	p.OpenContainerKind = ""
	p.OpenContainerStorage, p.ContainerSlots = nil, nil
	p.ContainerStateID++
	if conn == nil {
		return nil
	}
	if err := conn.WritePacket(protocol.NewBuilder(packetIDCloseContainerSC).VarInt(chestContainerID).Build()); err != nil {
		return err
	}
	return sendSetContainerContent(conn, p, p.ContainerStateID)
}
