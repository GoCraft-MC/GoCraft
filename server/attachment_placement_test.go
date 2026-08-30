package server

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func attachmentBedrockTestServer(t *testing.T) (*Server, *coreworld.World) {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	return &Server{world: w}, w
}

func TestBedrockAttachmentPlacementUsesSharedVanillaState(t *testing.T) {
	s, w := attachmentBedrockTestServer(t)
	p := player.New([16]byte{}, "builder", player.ClientEditionBedrock)
	p.Rotation.Yaw = 0
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	block := coreworld.Block{Namespace: "minecraft", Name: "oak_sign"}
	placed, valid := s.bedrockPlacementState(p, block, 1, 64, 0, intent.BlockInteractIntent{Face: 5})
	if !valid || placed.ResourceLocation() != "minecraft:oak_wall_sign" || placed.Properties["facing"] != "east" {
		t.Fatalf("placed=%+v valid=%v", placed, valid)
	}
}

func TestBedrockRailRequiresSupport(t *testing.T) {
	s, _ := attachmentBedrockTestServer(t)
	p := player.New([16]byte{}, "builder", player.ClientEditionBedrock)
	placed, valid := s.bedrockPlacementState(p, coreworld.Block{Namespace: "minecraft", Name: "rail"}, 5, 90, 5, intent.BlockInteractIntent{Face: 1})
	if valid {
		t.Fatalf("unsupported rail unexpectedly valid: %+v", placed)
	}
}
