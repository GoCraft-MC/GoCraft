package handler

import (
	"GoCraft/core/player"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

func sendExperience(conn *network.ClientConn, p *player.Player) error {
	level, total, progress := p.ExperienceSnapshot()
	return conn.WritePacket(protocol.NewBuilder(packetIDSetExperience).
		Float(progress).VarInt(level).VarInt(total).Build())
}

// SyncPlayerExperience refreshes the Java experience bar after canonical state changes.
func SyncPlayerExperience(p *player.Player, manager *session.Manager) {
	if p == nil || manager == nil {
		return
	}
	if current, ok := manager.Get(p.UUID); ok {
		_ = sendExperience(current.Conn, p)
	}
}
