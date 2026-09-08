package bedrock

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestLecternUsesLecternContainerType(t *testing.T) {
	got, ok := bedrockContainerType("minecraft:lectern")
	if !ok || got != protocol.ContainerTypeLectern {
		t.Fatalf("lectern container type = %d/%v, want %d/true", got, ok, protocol.ContainerTypeLectern)
	}
}
