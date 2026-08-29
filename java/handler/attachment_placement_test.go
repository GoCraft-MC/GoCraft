package handler

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestJavaAttachmentPlacementUsesSharedVanillaState(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	placed, handled, valid := coreworld.AttachmentPlacementState(w,
		coreworld.Block{Namespace: "minecraft", Name: "oak_sign"}, 1, 64, 0, 5, javaAttachmentRotation(0), false)
	if !handled || !valid || placed.ResourceLocation() != "minecraft:oak_wall_sign" || placed.Properties["facing"] != "east" {
		t.Fatalf("placed=%+v handled=%v valid=%v", placed, handled, valid)
	}
}

func TestJavaAttachmentRotationMatchesSignRotationRange(t *testing.T) {
	for _, yaw := range []float32{-720, -180, -90, 0, 90, 180, 720} {
		rotation := javaAttachmentRotation(yaw)
		if rotation < 0 || rotation > 15 {
			t.Fatalf("yaw %v produced rotation %d", yaw, rotation)
		}
	}
}
