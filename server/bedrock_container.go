package server

import (
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

func isBedrockGenericContainer(blockID string) bool {
	switch blockID {
	case "minecraft:chest", "minecraft:trapped_chest", "minecraft:barrel", "minecraft:ender_chest",
		"minecraft:hopper", "minecraft:dispenser", "minecraft:dropper", "minecraft:crafter":
		return true
	default:
		return false
	}
}

func bedrockGenericContainerSize(blockID string) int {
	switch blockID {
	case "minecraft:hopper":
		return 5
	case "minecraft:dispenser", "minecraft:dropper":
		return 9
	case "minecraft:crafter":
		return 9
	default:
		return 27
	}
}

func isBedrockWorkstation(blockID string) bool {
	switch blockID {
	case "minecraft:anvil", "minecraft:chipped_anvil", "minecraft:damaged_anvil",
		"minecraft:enchanting_table", "minecraft:grindstone", "minecraft:loom",
		"minecraft:smithing_table", "minecraft:stonecutter", "minecraft:brewing_stand",
		"minecraft:cartography_table", "minecraft:beacon":
		return true
	default:
		return false
	}
}

func bedrockWorkstationSlotCount(blockID string) int {
	switch blockID {
	case "minecraft:enchanting_table":
		return 2
	case "minecraft:loom", "minecraft:smithing_table":
		return 4
	case "minecraft:brewing_stand":
		return 5
	case "minecraft:beacon":
		return 1
	default:
		return 3
	}
}

func (s *Server) openBedrockWorkstation(p *player.Player, pos spatial.BlockPos, blockID string) {
	if p == nil || !isBedrockWorkstation(blockID) {
		return
	}
	p.OpenContainerID = 1
	p.OpenContainerKind = blockID
	p.OpenContainerPos = pos
	p.OpenContainerPartnerPos = spatial.BlockPos{}
	p.OpenContainerHasPartner = false
	p.ContainerSlots = make([]player.ItemStack, bedrockWorkstationSlotCount(blockID))
}

func (s *Server) returnBedrockWorkstationItems(p *player.Player) {
	if p == nil || !isBedrockWorkstation(p.OpenContainerKind) {
		return
	}
	for index, stack := range p.ContainerSlots {
		if stack.IsEmpty() || bedrockWorkstationOutputIndex(p.OpenContainerKind) == index {
			continue
		}
		if !p.GiveItem(stack) {
			if dropped := s.newDroppedItemForPlayer(p, stack, p.Position, index); dropped != nil && p.Dimension == dimensionOverworld {
				handler.BroadcastSpawnMob(dropped, s.sessions)
			}
		}
	}
}

func bedrockWorkstationOutputIndex(blockID string) int {
	switch blockID {
	case "minecraft:enchanting_table", "minecraft:brewing_stand", "minecraft:beacon":
		return -1
	case "minecraft:loom", "minecraft:smithing_table":
		return 3
	case "minecraft:stonecutter":
		return 1
	default:
		return 2
	}
}

func (s *Server) openBedrockGenericContainer(p *player.Player, pos spatial.BlockPos, blockID string) {
	if p == nil || s == nil || s.worldForPlayer(p) == nil {
		return
	}
	dimensionWorld := s.worldForPlayer(p)
	if blockID == "minecraft:chest" || blockID == "minecraft:trapped_chest" {
		handler.LoadChestContainerState(p, dimensionWorld, pos)
		return
	}
	p.OpenContainerID = 1
	p.OpenContainerKind = blockID
	p.OpenContainerPos = pos
	p.OpenContainerPartnerPos = spatial.BlockPos{}
	p.OpenContainerHasPartner = false
	p.ContainerSlots = make([]player.ItemStack, bedrockGenericContainerSize(blockID))
	for _, item := range dimensionWorld.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z)) {
		if item.Slot < 0 || item.Slot >= len(p.ContainerSlots) || item.ItemID == "" || item.Count <= 0 {
			continue
		}
		p.ContainerSlots[item.Slot] = player.ItemStack{ItemID: item.ItemID, Count: item.Count}
	}
}

func (s *Server) persistBedrockGenericContainer(p *player.Player) {
	if p == nil || s == nil || s.worldForPlayer(p) == nil || !isBedrockGenericContainer(p.OpenContainerKind) {
		return
	}
	if p.OpenContainerKind == "minecraft:chest" || p.OpenContainerKind == "minecraft:trapped_chest" {
		handler.PersistChestContents(p, s.worldForPlayer(p))
		return
	}
	items := make([]coreworld.ContainerItem, 0, len(p.ContainerSlots))
	for slot, stack := range p.ContainerSlots {
		if stack.IsEmpty() {
			continue
		}
		items = append(items, coreworld.ContainerItem{Slot: slot, ItemID: stack.ItemID, Count: stack.Count})
	}
	pos := p.OpenContainerPos
	s.worldForPlayer(p).SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), p.OpenContainerKind, items)
}
