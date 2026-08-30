package world

import "testing"

func TestBellRingDirectionUsesVanillaAttachmentRules(t *testing.T) {
	tests := []struct {
		name       string
		attachment string
		facing     string
		face       int32
		hitY       float32
		want       string
		valid      bool
	}{
		{"floor along axis", "floor", "north", 2, 0.5, "north", true},
		{"floor across axis", "floor", "north", 5, 0.5, "", false},
		{"wall across axis", "single_wall", "north", 5, 0.5, "east", true},
		{"wall along axis", "double_wall", "north", 3, 0.5, "", false},
		{"ceiling horizontal", "ceiling", "east", 4, 0.5, "west", true},
		{"vertical face", "ceiling", "east", 1, 0.5, "", false},
		{"hit above bell", "floor", "north", 2, 0.9, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block := Block{Namespace: "minecraft", Name: "bell", Properties: map[string]string{
				"attachment": test.attachment, "facing": test.facing, "powered": "false",
			}}
			got, valid := BellRingDirection(block, test.face, test.hitY)
			if got != test.want || valid != test.valid {
				t.Fatalf("direction=%q valid=%v, want %q/%v", got, valid, test.want, test.valid)
			}
		})
	}
}

func TestBellProjectileFaceUsesEntrySide(t *testing.T) {
	for _, test := range []struct {
		dx, dy, dz float64
		want       int32
	}{{1, 0, 0, 4}, {-1, 0, 0, 5}, {0, 0, 1, 2}, {0, 0, -1, 3}, {0, -1, 0, 1}} {
		if got := BellProjectileFace(test.dx, test.dy, test.dz); got != test.want {
			t.Fatalf("motion (%v,%v,%v) face=%d, want %d", test.dx, test.dy, test.dz, got, test.want)
		}
	}
}
