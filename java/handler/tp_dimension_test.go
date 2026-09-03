package handler

import (
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func TestTeleportSelfToPlayerChangesDimensionBeforePosition(t *testing.T) {
	issuer := player.New([16]byte{1}, "Issuer", player.ClientEditionJava)
	issuer.Dimension = 0
	target := player.New([16]byte{2}, "Target", player.ClientEditionBedrock)
	target.Dimension = 1
	target.Position = spatial.Vec3{X: 123.5, Y: 70, Z: -44.25}

	changedDimension := int32(-1)
	teleported := spatial.Vec3{}
	ctx := CommandContext{
		Player: issuer,
		Args:   []string{"Target"},
		FindPlayer: func(name string) *player.Player {
			if name == "Target" {
				return target
			}
			return nil
		},
		ChangeWorld: func(dimension int32) error {
			changedDimension = dimension
			issuer.Dimension = dimension
			return nil
		},
		TeleportTo: func(x, y, z float64) error {
			teleported = spatial.Vec3{X: x, Y: y, Z: z}
			return nil
		},
		Reply: func(string) error { return nil },
	}

	if err := cmdTp(ctx); err != nil {
		t.Fatalf("cmdTp returned error: %v", err)
	}
	if changedDimension != target.Dimension {
		t.Fatalf("dimension change = %d, want %d", changedDimension, target.Dimension)
	}
	if teleported != target.Position {
		t.Fatalf("teleport position = %+v, want %+v", teleported, target.Position)
	}
}

func TestTeleportSelfToPlayerSameDimensionSkipsWorldChange(t *testing.T) {
	issuer := player.New([16]byte{3}, "Issuer", player.ClientEditionJava)
	issuer.Dimension = 0
	target := player.New([16]byte{4}, "Target", player.ClientEditionBedrock)
	target.Dimension = 0
	target.Position = spatial.Vec3{X: 20, Y: 65, Z: 20}

	worldChanged := false
	ctx := CommandContext{
		Player:     issuer,
		Args:       []string{"Target"},
		FindPlayer: func(string) *player.Player { return target },
		ChangeWorld: func(int32) error {
			worldChanged = true
			return nil
		},
		TeleportTo: func(x, y, z float64) error { return nil },
		Reply:      func(string) error { return nil },
	}

	if err := cmdTp(ctx); err != nil {
		t.Fatalf("cmdTp returned error: %v", err)
	}
	if worldChanged {
		t.Fatal("same-dimension /tp unexpectedly changed world")
	}
}
