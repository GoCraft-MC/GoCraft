package world

import (
	"testing"

	"GoCraft/core/entity"
)

func TestVillagerProfessionsFollowExclusiveWorkstations(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "blast_furnace"})
	w.SetBlock(3, 64, 0, Block{Namespace: "minecraft", Name: "lectern"})

	first := entity.New(1, [16]byte{1}, entity.TypeVillager, 0, 64, 0)
	first.VillagerProfession = entity.VillagerProfessionNone
	first.VillagerLevel = 1
	second := entity.New(2, [16]byte{2}, entity.TypeVillager, 0, 64, 1)
	second.VillagerProfession = entity.VillagerProfessionNone
	second.VillagerLevel = 1
	w.Entities.Add(first)
	w.Entities.Add(second)

	if changed := w.RefreshVillagerProfessions(10); len(changed) != 2 {
		t.Fatalf("profession changes = %d, want 2", len(changed))
	}
	if !first.HasVillageWorkstation || !second.HasVillageWorkstation || first.VillageWorkstation == second.VillageWorkstation {
		t.Fatalf("workstation claims are not exclusive: first=%+v second=%+v", first, second)
	}
	if first.VillagerProfession == second.VillagerProfession {
		t.Fatalf("different job sites produced the same profession: %s", first.VillagerProfession)
	}
}

func TestUntradedVillagerChangesProfessionButTradedVillagerKeepsIt(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	position := struct{ X, Y, Z int }{1, 64, 0}
	w.SetBlock(position.X, position.Y, position.Z, Block{Namespace: "minecraft", Name: "blast_furnace"})
	villager := entity.New(3, [16]byte{3}, entity.TypeVillager, 0, 64, 0)
	villager.VillagerProfession = entity.VillagerProfessionNone
	villager.VillagerLevel = 1
	w.Entities.Add(villager)
	w.RefreshVillagerProfessions(10)
	if villager.VillagerProfession != entity.VillagerProfessionArmorer {
		t.Fatalf("profession = %s, want armorer", villager.VillagerProfession)
	}

	w.SetBlock(position.X, position.Y, position.Z, Block{Namespace: "minecraft", Name: "lectern"})
	w.RefreshVillagerProfessions(10)
	if villager.VillagerProfession != entity.VillagerProfessionLibrarian {
		t.Fatalf("untraded profession = %s, want librarian after workstation change", villager.VillagerProfession)
	}

	villager.VillagerHasTraded = true
	villager.VillagerExperience = 1
	w.SetBlock(position.X, position.Y, position.Z, Air)
	w.RefreshVillagerProfessions(10)
	if villager.VillagerProfession != entity.VillagerProfessionLibrarian {
		t.Fatalf("traded villager lost locked profession: %s", villager.VillagerProfession)
	}
}
