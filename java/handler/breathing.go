package handler

import (
	"GoCraft/core/player"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

func buildPlayerAirSupply(p *player.Player) *protocol.Packet {
	return protocol.NewBuilder(packetIDSetEntityData).
		VarInt(p.EntityID).
		Byte(1).   // Entity air supply metadata index.
		VarInt(1). // VarInt metadata serializer.
		VarInt(p.AirSupplySnapshot()).
		Byte(0xff).
		Build()
}

// SyncPlayerAirSupply refreshes the local Java bubble meter.
func SyncPlayerAirSupply(p *player.Player, manager *session.Manager) {
	if p == nil || manager == nil {
		return
	}
	if current, ok := manager.Get(p.UUID); ok && current.Conn != nil {
		_ = current.Conn.WritePacket(buildPlayerAirSupply(p))
	}
}
