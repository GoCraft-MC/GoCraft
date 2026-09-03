package bedrock

import (
	"testing"

	"GoCraft/core/spatial"
)

func TestBedrockRemotePlayerTeleportUsesActorRespawn(t *testing.T) {
	origin := spatial.Vec3{X: 10, Y: 64, Z: 10}

	if bedrockRemotePlayerNeedsRespawn(origin, spatial.Vec3{X: 17.9, Y: 64, Z: 10}) {
		t.Fatal("small movement should stay on the normal movement path")
	}
	if !bedrockRemotePlayerNeedsRespawn(origin, spatial.Vec3{X: 18.1, Y: 64, Z: 10}) {
		t.Fatal("large horizontal jump should respawn the remote actor")
	}
	if !bedrockRemotePlayerNeedsRespawn(origin, spatial.Vec3{X: 10, Y: 72.1, Z: 10}) {
		t.Fatal("large vertical jump should respawn the remote actor")
	}
}
