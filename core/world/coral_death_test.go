package world

import "testing"

func TestCoralSurvivesNextToWater(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "tube_coral_block"})
	w.SetBlock(1, 64, 0, MakeFluid("minecraft:water", 0))
	if _, died := w.ApplyCoralDeath(0, 64, 0); died {
		t.Fatal("coral touching water should not die")
	}
	if got := w.GetBlock(0, 64, 0).ResourceLocation(); got != "minecraft:tube_coral_block" {
		t.Fatalf("coral = %s, want it to survive", got)
	}
}

func TestCoralDiesWithoutWater(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "fire_coral_fan"})
	change, died := w.ApplyCoralDeath(0, 64, 0)
	if !died {
		t.Fatal("coral out of water should die")
	}
	if change.Block.ResourceLocation() != "minecraft:dead_fire_coral_fan" {
		t.Fatalf("dead coral = %s, want minecraft:dead_fire_coral_fan", change.Block.ResourceLocation())
	}
}

func TestWaterloggedCoralSurvives(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	coral := Block{Namespace: "minecraft", Name: "brain_coral", Properties: map[string]string{"waterlogged": "true"}}
	w.SetBlock(0, 64, 0, coral)
	if _, died := w.ApplyCoralDeath(0, 64, 0); died {
		t.Fatal("waterlogged coral should survive")
	}
}

func TestPlacingCoralSchedulesDeathCheck(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "horn_coral"})
	due := w.BlockPhysics.DrainDue(w.PhysicsTime() + coralDeathDelay)
	found := false
	for _, update := range due {
		if update.Kind == UpdateCoralDeath && update.X == 0 && update.Y == 64 && update.Z == 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("placing coral did not schedule a death check")
	}
}
