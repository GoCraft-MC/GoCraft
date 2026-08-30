package bedrock

import (
	"GoCraft/core/spatial"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func bedrockBellPackets(position spatial.BlockPos, direction string) [2]packet.Packet {
	blockPosition := protocol.BlockPos{position.X, position.Y, position.Z}
	return [2]packet.Packet{
		&packet.BlockActorData{Position: blockPosition, NBTData: map[string]any{
			"id": "Bell", "x": position.X, "y": position.Y, "z": position.Z,
			"Direction": bedrockBellDirection(direction), "Ringing": uint8(1), "Ticks": int32(0),
		}},
		&packet.LevelSoundEvent{
			SoundType: packet.SoundEventBell,
			Position:  mgl32.Vec3{float32(position.X) + 0.5, float32(position.Y) + 0.5, float32(position.Z) + 0.5},
			ExtraData: -1,
		},
	}
}

func bedrockBellDirection(direction string) int32 {
	switch direction {
	case "west":
		return 1
	case "north":
		return 2
	case "east":
		return 3
	default:
		return 0
	}
}

// BroadcastBellRing mirrors one canonical bell action to Bedrock viewers in
// the same dimension using a transient block actor animation and bell sound.
func (l *Listener) BroadcastBellRing(dimension int32, position spatial.BlockPos, direction string) {
	if l == nil {
		return
	}
	packets := bedrockBellPackets(position, direction)
	l.sessionsMu.RLock()
	sessions := make([]*bedrockSession, 0, len(l.sessions))
	for _, current := range l.sessions {
		if current.dimension.Load() == dimension {
			sessions = append(sessions, current)
		}
	}
	l.sessionsMu.RUnlock()
	for _, current := range sessions {
		for _, event := range packets {
			_ = current.conn.WritePacket(event)
		}
	}
}
