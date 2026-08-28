package handler

import (
	"GoCraft/core/player"
	"GoCraft/java/network"
)

func SyncCommandPermissions(conn *network.ClientConn, player *player.Player, dispatcher *Dispatcher) error {
	return conn.WritePacket(buildCommandsPacket(func(name string) bool { return dispatcher.CanUse(player, name) }))
}
