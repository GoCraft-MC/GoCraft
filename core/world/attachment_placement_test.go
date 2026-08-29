package world

import "testing"

func attachmentTestWorld(t *testing.T) *World {
	t.Helper()
	w := New(&FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	return w
}

func TestAttachmentPlacementRejectsInvalidFacesAndSupport(t *testing.T) {
	w := attachmentTestWorld(t)
	stone := Block{Namespace: "minecraft", Name: "stone"}
	w.SetBlock(0, 64, 0, stone)

	tests := []struct {
		name    string
		block   Block
		face    int32
		x, y, z int
	}{
		{"sign ceiling", Block{Namespace: "minecraft", Name: "oak_sign"}, 0, 0, 63, 0},
		{"banner ceiling", Block{Namespace: "minecraft", Name: "white_banner"}, 0, 0, 63, 0},
		{"skull ceiling", Block{Namespace: "minecraft", Name: "skeleton_skull"}, 0, 0, 63, 0},
		{"coral fan ceiling", Block{Namespace: "minecraft", Name: "tube_coral_fan"}, 0, 0, 63, 0},
		{"ladder floor", Block{Namespace: "minecraft", Name: "ladder"}, 1, 0, 65, 0},
		{"rail no floor", Block{Namespace: "minecraft", Name: "rail"}, 1, 5, 65, 0},
		{"flower pot no floor", Block{Namespace: "minecraft", Name: "flower_pot"}, 1, 5, 65, 0},
		{"candle no floor", Block{Namespace: "minecraft", Name: "candle"}, 1, 5, 65, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handled, valid := AttachmentPlacementState(w, tc.block, tc.x, tc.y, tc.z, tc.face, 0, false)
			if !handled || valid {
				t.Fatalf("handled=%v valid=%v, want handled invalid", handled, valid)
			}
		})
	}
}

func TestAttachmentPlacementBuildsWallVariants(t *testing.T) {
	w := attachmentTestWorld(t)
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	cases := []struct {
		name, item, want string
	}{
		{"sign", "oak_sign", "minecraft:oak_wall_sign"},
		{"hanging sign", "oak_hanging_sign", "minecraft:oak_wall_hanging_sign"},
		{"banner", "white_banner", "minecraft:white_wall_banner"},
		{"skull", "skeleton_skull", "minecraft:skeleton_wall_skull"},
		{"coral", "tube_coral_fan", "minecraft:tube_coral_wall_fan"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			placed, handled, valid := AttachmentPlacementState(w, Block{Namespace: "minecraft", Name: tc.item}, 1, 64, 0, 5, 3, false)
			if !handled || !valid || placed.ResourceLocation() != tc.want || placed.Properties["facing"] != "east" {
				t.Fatalf("placed=%+v handled=%v valid=%v, want %s facing east", placed, handled, valid, tc.want)
			}
		})
	}
}

func TestAttachmentPlacementFloorCeilingAndLantern(t *testing.T) {
	w := attachmentTestWorld(t)
	stone := Block{Namespace: "minecraft", Name: "stone"}
	w.SetBlock(0, 64, 0, stone)
	w.SetBlock(2, 66, 0, stone)

	sign, _, valid := AttachmentPlacementState(w, Block{Namespace: "minecraft", Name: "oak_sign"}, 0, 65, 0, 1, 7, false)
	if !valid || sign.Properties["rotation"] != "7" {
		t.Fatalf("standing sign = %+v valid=%v", sign, valid)
	}
	hanging, _, valid := AttachmentPlacementState(w, Block{Namespace: "minecraft", Name: "oak_hanging_sign"}, 2, 65, 0, 0, 9, false)
	if !valid || hanging.Properties["rotation"] != "9" {
		t.Fatalf("hanging sign = %+v valid=%v", hanging, valid)
	}
	lantern, _, valid := AttachmentPlacementState(w, Block{Namespace: "minecraft", Name: "lantern"}, 2, 65, 0, 0, 0, false)
	if !valid || lantern.Properties["hanging"] != "true" {
		t.Fatalf("lantern = %+v valid=%v", lantern, valid)
	}
}

func TestBreakUnsupportedDecorativeAttachments(t *testing.T) {
	w := attachmentTestWorld(t)
	stone := Block{Namespace: "minecraft", Name: "stone"}
	w.SetBlock(0, 64, 0, stone)
	attachments := []Block{
		{Namespace: "minecraft", Name: "oak_wall_sign", Properties: map[string]string{"facing": "east"}},
		{Namespace: "minecraft", Name: "white_wall_banner", Properties: map[string]string{"facing": "east"}},
		{Namespace: "minecraft", Name: "ladder", Properties: map[string]string{"facing": "east"}},
		{Namespace: "minecraft", Name: "player_wall_head", Properties: map[string]string{"facing": "east"}},
		{Namespace: "minecraft", Name: "tube_coral_wall_fan", Properties: map[string]string{"facing": "east"}},
	}
	for index, attachment := range attachments {
		z := index * 2
		w.SetBlock(0, 64, z, stone)
		w.SetBlock(1, 64, z, attachment)
		w.SetBlock(0, 64, z, Air)
		changes := w.BreakUnsupportedAttachmentsAround(0, 64, z)
		if len(changes) != 1 || !w.GetBlock(1, 64, z).IsAir() {
			t.Fatalf("%s support removal changes=%+v block=%+v", attachment.ResourceLocation(), changes, w.GetBlock(1, 64, z))
		}
	}
}
