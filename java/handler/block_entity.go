package handler

import (
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

func buildBlockEntityData(entity coreworld.BlockEntity) *protocol.Packet {
	typeID, ok := javaworld.BlockEntityTypeID(entity.Type)
	data := javaworld.BlockEntityNetworkData(entity)
	if !ok || len(data) == 0 {
		return nil
	}
	position := spatial.BlockPos{X: int32(entity.X), Y: int32(entity.Y), Z: int32(entity.Z)}
	return protocol.NewBuilder(packetIDBlockEntityData).
		Long(position.Encode()).
		VarInt(typeID).
		Bytes(data).
		Build()
}

// BroadcastBlockEntityDataInDimension mirrors one canonical block-entity
// mutation to Java viewers in the same dimension.
func BroadcastBlockEntityDataInDimension(entity coreworld.BlockEntity, mgr *session.Manager, dimension int32) {
	if mgr == nil {
		return
	}
	pkt := buildBlockEntityData(entity)
	if pkt == nil {
		return
	}
	for _, current := range mgr.SnapshotAll() {
		if current.Player == nil || current.Player.Dimension != dimension {
			continue
		}
		_ = current.Conn.WritePacket(pkt)
	}
}
