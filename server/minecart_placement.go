package server

import (
	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/handler"
)

func canonicalMinecartType(item string) (corentity.EntityType, bool) {
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

func (s *Server) placeBedrockMinecart(p *player.Player, position spatial.BlockPos) bool {
	entityType, minecart := canonicalMinecartType(p.HeldItem().ItemID)
	if !minecart || s.game == nil || s.bedrockWorld() == nil {
		return false
	}
	x, y, z := int(position.X), int(position.Y), int(position.Z)
	if !isMinecartRail(s.bedrockWorld().GetBlock(x, y, z).ResourceLocation()) {
		return true
	}
	entity := corentity.New(s.game.NextEntityID(), [16]byte{}, entityType,
		float64(x)+0.5, float64(y)+0.0625, float64(z)+0.5)
	entity.Yaw = p.Rotation.Yaw
	s.bedrockWorld().Entities.Add(entity)
	handler.BroadcastSpawnMob(entity, s.sessions)
	s.consumeBedrockHeldItem(p, 1)
	return true
}
