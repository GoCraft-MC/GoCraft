package world

import "testing"

func TestBellPlacementMatchesVanillaAttachments(t *testing.T) {
	w := attachmentTestWorld(t)
	stone := Block{Namespace: "minecraft", Name: "stone"}
	bell := Block{Namespace: "minecraft", Name: "bell"}

	w.SetBlock(0, 64, 0, stone)
	floor, ok := bellPlacementState(w, bell, 0, 65, 0, 1, 8)
	if !ok || floor.Properties["attachment"] != "floor" || floor.Properties["facing"] != "south" {
		t.Fatalf("floor bell = %+v ok=%v", floor, ok)
	}

	w.SetBlock(2, 66, 0, stone)
	ceiling, ok := bellPlacementState(w, bell, 2, 65, 0, 0, 0)
	if !ok || ceiling.Properties["attachment"] != "ceiling" || ceiling.Properties["facing"] != "north" {
		t.Fatalf("ceiling bell = %+v ok=%v", ceiling, ok)
	}

	w.SetBlock(4, 64, 0, stone)
	single, ok := bellPlacementState(w, bell, 5, 64, 0, 5, 0)
	if !ok || single.Properties["attachment"] != "single_wall" || single.Properties["facing"] != "west" {
		t.Fatalf("single wall bell = %+v ok=%v", single, ok)
	}

	w.SetBlock(6, 64, 0, stone)
	double, ok := bellPlacementState(w, bell, 5, 64, 0, 5, 0)
	if !ok || double.Properties["attachment"] != "double_wall" || double.Properties["facing"] != "west" {
		t.Fatalf("double wall bell = %+v ok=%v", double, ok)
	}
}

func TestBellNeighborSupportTransitions(t *testing.T) {
	w := attachmentTestWorld(t)
	stone := Block{Namespace: "minecraft", Name: "stone"}
	w.SetBlock(0, 64, 0, stone)
	w.SetBlock(2, 64, 0, stone)
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "bell", Properties: map[string]string{
		"attachment": "double_wall", "facing": "west", "powered": "false",
	}})

	w.SetBlock(2, 64, 0, Air)
	changes := w.BreakUnsupportedAttachmentsAround(2, 64, 0)
	got := w.GetBlock(1, 64, 0)
	if len(changes) != 1 || got.Properties["attachment"] != "single_wall" || got.Properties["facing"] != "west" {
		t.Fatalf("double -> single changes=%+v bell=%+v", changes, got)
	}

	w.SetBlock(0, 64, 0, Air)
	changes = w.BreakUnsupportedAttachmentsAround(0, 64, 0)
	if len(changes) != 1 || !w.GetBlock(1, 64, 0).IsAir() {
		t.Fatalf("single -> air changes=%+v bell=%+v", changes, w.GetBlock(1, 64, 0))
	}
}
