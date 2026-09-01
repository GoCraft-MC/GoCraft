package handler

import (
	"GoCraft/core/player"
	"GoCraft/java/network"
)

func SyncCommandPermissions(conn *network.ClientConn, player *player.Player, dispatcher *Dispatcher) error {
	return conn.WritePacket(buildCommandsPacket(dispatcher.CommandTree(player)))
}
