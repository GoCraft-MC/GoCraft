package bedrock

import (
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestCompletedFoodUseEmitsEatAndBurpSounds(t *testing.T) {
	p := player.New([16]byte{77}, "hungry", player.ClientEditionBedrock)
	p.Position = spatial.Vec3{X: 4.5, Y: 65, Z: -3.5}
	events := completedFoodSoundEvents(p, 91)
	if events[0].SoundType != packet.SoundEventEat || events[1].SoundType != packet.SoundEventBurp {
		t.Fatalf("food sounds = %q/%q, want eat/burp", events[0].SoundType, events[1].SoundType)
	}
	for _, event := range events {
		if event.EntityType != "minecraft:player" || event.EntityUniqueID != 91 || event.ExtraData != -1 {
			t.Fatalf("invalid food sound descriptor: %+v", event)
		}
		if event.Position[0] != 4.5 || event.Position[1] != 65 || event.Position[2] != -3.5 {
			t.Fatalf("food sound position = %v", event.Position)
		}
	}
}
