package handler

import (
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

const furnaceContainerID int32 = 1

func openFurnace(p *player.Player, conn *network.ClientConn, w *coreworld.World, pos spatial.BlockPos, blockID string) error {
	if p.OpenContainerKind == "minecraft:crafting_table" {
		returnCraftingGrid(p)
	}
	p.OpenContainerID = furnaceContainerID
	p.OpenContainerKind = blockID
	p.OpenContainerPos = pos
	p.OpenContainerPartnerPos = spatial.BlockPos{}
	p.OpenContainerHasPartner = false
	p.ContainerSlots = make([]player.ItemStack, 3)
	for _, item := range w.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z)) {
		if item.Slot >= 0 && item.Slot < 3 && item.ItemID != "" && item.Count > 0 {
			p.ContainerSlots[item.Slot] = player.ItemStack{ItemID: item.ItemID, Count: item.Count}
		}
	}
	p.ContainerStateID++
	if err := sendOpenScreen(conn, furnaceContainerID, containerMenuType(blockID), containerTitle(blockID)); err != nil {
		return err
	}
	return SyncFurnaceContainer(conn, p, 0, 0, 0, 0)
}

// SyncFurnaceContainer sends the three furnace slots, player storage, cursor,
// and the four vanilla furnace progress properties.
func SyncFurnaceContainer(conn *network.ClientConn, p *player.Player, cookTime, burnTime, burnDuration, cookDuration int) error {
	if conn == nil || p == nil || !IsFurnaceContainer(p.OpenContainerKind) {
		return nil
	}
	if err := sendFurnaceContainerContent(conn, p); err != nil {
		return err
	}
	properties := [4]int{burnTime, burnDuration, cookTime, cookDuration}
	for property, value := range properties {
		if err := conn.WritePacket(protocol.NewBuilder(packetIDSetContainerData).
			VarInt(furnaceContainerID).Short(int16(property)).Short(int16(value)).Build()); err != nil {
			return err
		}
	}
	return nil
}

func sendFurnaceContainerContent(conn *network.ClientConn, p *player.Player) error {
	b := protocol.NewBuilder(packetIDSetContainerContent).
		VarInt(furnaceContainerID).
		VarInt(p.ContainerStateID).
		VarInt(39)
	for slot := 0; slot < 3; slot++ {
		if slot < len(p.ContainerSlots) {
			encodeSlot(b, p.ContainerSlots[slot])
		} else {
			encodeSlot(b, player.ItemStack{})
		}
	}
	for slot := 9; slot < player.HotbarStart; slot++ {
		encodeSlot(b, p.Inventory[slot])
	}
	for slot := player.HotbarStart; slot < player.HotbarStart+9; slot++ {
		encodeSlot(b, p.Inventory[slot])
	}
	encodeSlot(b, p.CarriedItem)
	return conn.WritePacket(b.Build())
}

func handleFurnaceClick(p *player.Player, w *coreworld.World, slot int, button byte, mode int32) {
	switch mode {
	case 0:
		clickFurnaceSlot(p, slot, button)
	case 1:
		shiftFurnaceSlot(p, slot)
	}
	p.ContainerStateID++
	persistFurnaceContents(p, w)
}

func clickFurnaceSlot(p *player.Player, slot int, button byte) {
	if slot < 0 || slot >= 39 {
		return
	}
	target := furnaceContainerSlot(p, slot)
	if target == nil {
		return
	}
	if slot == 2 {
		if target.IsEmpty() {
			return
		}
		if p.CarriedItem.IsEmpty() {
			p.CarriedItem, *target = *target, player.ItemStack{}
		} else if p.CarriedItem.ItemID == target.ItemID && p.CarriedItem.Damage == target.Damage &&
			p.CarriedItem.Count+target.Count <= player.MaxStackSize(target.ItemID) {
			p.CarriedItem.Count += target.Count
			*target = player.ItemStack{}
		}
		return
	}
	beforeTarget, beforeCarried := *target, p.CarriedItem
	clickChestSlot(p, slot, button)
	if slot < 2 && !canPlaceJavaFurnaceSlot(slot, *target) {
		*target, p.CarriedItem = beforeTarget, beforeCarried
	}
}

func shiftFurnaceSlot(p *player.Player, slot int) {
	target := furnaceContainerSlot(p, slot)
	if target == nil || target.IsEmpty() {
		return
	}
	if slot < 3 {
		inventory := p.Inventory
		if addStackToInventory(&inventory, *target) {
			p.Inventory = inventory
			*target = player.ItemStack{}
		}
		return
	}
	destination := 0
	if CanPlaceFurnaceFuelSlot(target.ItemID) {
		destination = 1
	}
	if addStackToFurnaceSlot(&p.ContainerSlots[destination], *target) {
		*target = player.ItemStack{}
	}
}

func addStackToFurnaceSlot(destination *player.ItemStack, source player.ItemStack) bool {
	if destination == nil || source.IsEmpty() {
		return false
	}
	if destination.IsEmpty() {
		if source.Count > player.MaxStackSize(source.ItemID) {
			return false
		}
		*destination = source
		return true
	}
	if destination.ItemID != source.ItemID || destination.Damage != source.Damage ||
		destination.Count+source.Count > player.MaxStackSize(destination.ItemID) {
		return false
	}
	destination.Count += source.Count
	return true
}

func canPlaceJavaFurnaceSlot(slot int, stack player.ItemStack) bool {
	if stack.IsEmpty() {
		return true
	}
	switch slot {
	case 0:
		return true
	case 1:
		return CanPlaceFurnaceFuelSlot(stack.ItemID)
	case 2:
		return false
	default:
		return true
	}
}

func furnaceContainerSlot(p *player.Player, slot int) *player.ItemStack {
	if p == nil || len(p.ContainerSlots) != 3 {
		return nil
	}
	switch {
	case slot >= 0 && slot < 3:
		return &p.ContainerSlots[slot]
	case slot >= 3 && slot < 30:
		return &p.Inventory[9+slot-3]
	case slot >= 30 && slot < 39:
		return &p.Inventory[player.HotbarStart+slot-30]
	default:
		return nil
	}
}

func persistFurnaceContents(p *player.Player, w *coreworld.World) {
	if p == nil || w == nil || !IsFurnaceContainer(p.OpenContainerKind) {
		return
	}
	items := make([]coreworld.ContainerItem, 0, 3)
	for slot := 0; slot < 3 && slot < len(p.ContainerSlots); slot++ {
		if stack := p.ContainerSlots[slot]; !stack.IsEmpty() {
			items = append(items, coreworld.ContainerItem{Slot: slot, ItemID: stack.ItemID, Count: stack.Count})
		}
	}
	pos := p.OpenContainerPos
	w.SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), p.OpenContainerKind, items)
}
