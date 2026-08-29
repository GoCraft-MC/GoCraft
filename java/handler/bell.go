package handler

import (
	"GoCraft/core/spatial"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

func buildBellRingPackets(position spatial.BlockPos, direction string) []*protocol.Packet {
	bellID, ok := javaworld.BlockTypeID("minecraft:bell")
	if !ok {
		return nil
	}
	blockEvent := protocol.NewBuilder(packetIDBlockAction).
		Long(position.Encode()).
		Byte(1).
		Byte(javaBellDirection(direction)).
		VarInt(bellID).
		Build()
	sound := buildSoundAt("minecraft:block.bell.use", soundCategoryBlocks,
		float64(position.X)+0.5, float64(position.Y)+0.5, float64(position.Z)+0.5, 2, 1)
	if sound == nil {
		return []*protocol.Packet{blockEvent}
	}
	return []*protocol.Packet{blockEvent, sound}
}

func javaBellDirection(direction string) byte {
	switch direction {
	case "south":
		return 3
	case "west":
		return 4
	case "east":
		return 5
	default:
		return 2
	}
}

// BroadcastBellRing sends the transient bell animation and sound to Java
// viewers in the bell's dimension. The permanent block state is untouched.
func BroadcastBellRing(position spatial.BlockPos, direction string, dimension int32, mgr *session.Manager) {
	if mgr == nil {
		return
	}
	packets := buildBellRingPackets(position, direction)
	for _, current := range mgr.SnapshotAll() {
		if current.Player == nil || current.Player.Dimension != dimension {
			continue
		}
		for _, packet := range packets {
			_ = current.Conn.WritePacket(packet)
		}
	}
}
