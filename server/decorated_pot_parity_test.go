package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestBedrockDecoratedPotPlacementPreservesDecorations(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{81}, "bedrock-potter", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	p.Rotation.Yaw = 0
	decorations := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:heart_pottery_sherd"}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:decorated_pot", Count: 1, PotDecorations: decorations}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.SetBlock(0, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	s := &Server{game: g, world: w, sessions: session.NewManager()}
	if !s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
		PlayerUUID: p.UUID, Position: spatial.BlockPos{X: 0, Y: 63, Z: 0}, Face: 1, ClickX: 0.5, ClickY: 1, ClickZ: 0.5,
	}, w.GetBlock(0, 63, 0)) {
		t.Fatal("decorated pot placement was not handled")
	}
	pot := w.GetBlock(0, 64, 0)
	if pot.ResourceLocation() != "minecraft:decorated_pot" || pot.Properties["cracked"] != "false" {
		t.Fatalf("placed pot state = %#v", pot)
	}
	if got := w.DecoratedPotDecorations(0, 64, 0); got != decorations {
		t.Fatalf("placed decorations = %#v, want %#v", got, decorations)
	}
}

func TestSnowballImpactQueuesZeroDamageReaction(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	pig := corentity.New(991, [16]byte{9, 9, 1}, corentity.TypePig, 0, 64, 0)
	w.Entities.Add(pig)
	if !w.QueueEntityImpactFrom(pig.EntityID, -1, 0) {
		t.Fatal("zero-damage impact was not queued")
	}
	events := w.DrainEntityDamage()
	event, ok := events[pig.EntityID]
	if !ok || event.Amount != 0 || !event.HasSource {
		t.Fatalf("impact event = %#v, present=%v", event, ok)
	}
}
