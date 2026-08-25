package handler

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestUseEnderEyeTargetsNearestStronghold(t *testing.T) {
	w := coreworld.New(coreworld.NewOverworldGenerator(0), nil, false)
	defer w.Close()
	p := player.New([16]byte{5}, "finder", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.EntityID = 9
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:ender_eye", Count: 2}
	UseEnderEye(p, w, session.NewManager(), func() int32 { return 44 })
	entities := w.Entities.Snapshot()
	if len(entities) != 1 || entities[0].Type != corentity.TypeEyeOfEnder {
		t.Fatalf("spawned eye = %+v", entities)
	}
	eye := entities[0]
	if !eye.HasEyeTarget || eye.EyeTarget.X != -184 || eye.EyeTarget.Z != -1784 {
		t.Fatalf("eye target = %+v, present=%v", eye.EyeTarget, eye.HasEyeTarget)
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 1 {
		t.Fatalf("remaining eyes = %d, want 1", got)
	}
}
