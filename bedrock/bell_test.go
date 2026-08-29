package bedrock

import (
	"testing"

	"GoCraft/core/spatial"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestBedrockBellUsesBlockActorAnimationAndNativeSound(t *testing.T) {
	packets := bedrockBellPackets(spatial.BlockPos{X: 1, Y: 64, Z: 2}, "east")
	actor, ok := packets[0].(*packet.BlockActorData)
	if !ok || actor.NBTData["id"] != "Bell" || actor.NBTData["Direction"] != int32(3) || actor.NBTData["Ringing"] != uint8(1) {
		t.Fatalf("block actor=%#v", packets[0])
	}
	sound, ok := packets[1].(*packet.LevelSoundEvent)
	if !ok || sound.SoundType != packet.SoundEventBell {
		t.Fatalf("sound=%#v", packets[1])
	}
}
