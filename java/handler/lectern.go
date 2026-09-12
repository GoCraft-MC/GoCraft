package handler

import (
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

func openLectern(p *player.Player, conn *network.ClientConn, pos spatial.BlockPos, entity coreworld.BlockEntity) error {
	p.OpenContainerID = chestContainerID
	p.OpenContainerKind = "minecraft:lectern"
	p.OpenContainerPos = pos
	p.ContainerStateID++
	p.ContainerSlots = nil
	for _, item := range entity.Items {
		if item.Slot == 0 {
			p.ContainerSlots = []player.ItemStack{item.Stack()}
			break
		}
	}
	if conn == nil {
		return nil
	}
	if err := sendOpenScreen(conn, chestContainerID, containerMenuType("minecraft:lectern"), "Lectern"); err != nil {
		return err
	}
	b := protocol.NewBuilder(packetIDSetContainerContent).
		VarInt(chestContainerID).VarInt(p.ContainerStateID).VarInt(1)
	if len(p.ContainerSlots) == 1 {
		encodeSlot(b, p.ContainerSlots[0])
	} else {
		encodeSlot(b, player.ItemStack{})
	}
	encodeSlot(b, p.CarriedItem)
	if err := conn.WritePacket(b.Build()); err != nil {
		return err
	}
	return conn.WritePacket(protocol.NewBuilder(packetIDSetContainerData).
		VarInt(chestContainerID).Short(0).Short(int16(entity.LecternPage)).Build())
}
