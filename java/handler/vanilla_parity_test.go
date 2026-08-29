package handler

import (
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func TestJavaLeverPlacementAllAttachmentFaces(t *testing.T) {
	tests := []struct {
		face                 int32
		wantFace, wantFacing string
	}{
		{0, "ceiling", "south"}, {1, "floor", "south"},
		{2, "wall", "north"}, {3, "wall", "south"},
		{4, "wall", "west"}, {5, "wall", "east"},
	}
	for _, tc := range tests {
		w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
		x, y, z := 0, 64, 0
		off := faceOffset[tc.face]
		w.SetBlock(x-int(off[0]), y-int(off[1]), z-int(off[2]), coreworld.Block{Namespace: "minecraft", Name: "stone"})
		got, ok := javaButtonPlacementState(coreworld.Block{Namespace: "minecraft", Name: "lever"}, tc.face, 0, w, x, y, z)
		if !ok {
			t.Fatalf("face %d rejected", tc.face)
		}
		if got.Properties["face"] != tc.wantFace || got.Properties["facing"] != tc.wantFacing {
			t.Fatalf("face %d => face=%q facing=%q, want %q/%q", tc.face, got.Properties["face"], got.Properties["facing"], tc.wantFace, tc.wantFacing)
		}
		w.Close()
	}
}

func TestJavaDoorPlacementUsesSharedVanillaHinge(t *testing.T) {
	p := player.New([16]byte{}, "door", player.ClientEditionJava)
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	p.Rotation.Yaw = 180 // north in GoCraft's placement convention
	if !placeDoorBlock(p, 0, 64, 0, "minecraft:oak_door", 0.2, 0.5, w, nil) {
		t.Fatal("door placement rejected")
	}
	lower := w.GetBlock(0, 64, 0)
	upper := w.GetBlock(0, 65, 0)
	if lower.Properties["hinge"] != coreworld.DoorHinge(w, 0, 64, 0, lower.Properties["facing"], 0.2, 0.5) {
		t.Fatalf("lower hinge=%q", lower.Properties["hinge"])
	}
	if upper.Properties["hinge"] != lower.Properties["hinge"] || upper.Properties["facing"] != lower.Properties["facing"] {
		t.Fatalf("upper/lower mismatch: %+v %+v", lower.Properties, upper.Properties)
	}
}
