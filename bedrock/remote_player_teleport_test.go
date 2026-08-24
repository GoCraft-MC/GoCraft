package bedrock

import (
	"testing"

	"GoCraft/core/spatial"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestBedrockRemotePlayerMoveMode(t *testing.T) {
	origin := spatial.Vec3{X: 10, Y: 64, Z: 10}

	if got := bedrockRemotePlayerMoveMode(origin, spatial.Vec3{X: 17.9, Y: 64, Z: 10}); got != byte(packet.MoveModeNormal) {
		t.Fatalf("small movement should stay normal, got mode %d", got)
	}
	if got := bedrockRemotePlayerMoveMode(origin, spatial.Vec3{X: 18.1, Y: 64, Z: 10}); got != byte(packet.MoveModeTeleport) {
		t.Fatalf("large horizontal jump should use teleport mode, got mode %d", got)
	}
	if got := bedrockRemotePlayerMoveMode(origin, spatial.Vec3{X: 10, Y: 72.1, Z: 10}); got != byte(packet.MoveModeTeleport) {
		t.Fatalf("large vertical jump should use teleport mode, got mode %d", got)
	}
}
