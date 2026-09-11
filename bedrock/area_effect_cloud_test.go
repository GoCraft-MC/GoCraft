package bedrock

import (
	"math"
	"testing"

	corentity "GoCraft/core/entity"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestBedrockAreaEffectCloudMetadata(t *testing.T) {
	cloud := corentity.New(71, [16]byte{}, corentity.TypeAreaEffectCloud, 0, 64, 0)
	cloud.CloudRadius = 2.75
	metadata := (&Listener{}).bedrockEntityMetadata(nil, cloud)
	if got := metadata[protocol.EntityDataKeyDataRadius]; got != float32(2.75) {
		t.Fatalf("cloud radius metadata = %v", got)
	}
	if got := metadata[protocol.EntityDataKeyDataDuration]; got != int32(math.MaxInt32) {
		t.Fatalf("cloud duration metadata = %v", got)
	}
	if metadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasGravity) ||
		metadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagHasCollision) {
		t.Fatal("stationary cloud received gravity or collision flags")
	}
}
