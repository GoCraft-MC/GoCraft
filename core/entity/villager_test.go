package entity

import "testing"

func TestVillagerCanTradeMatchesPumpkinInteractionRules(t *testing.T) {
	villager := New(1, [16]byte{}, TypeVillager, 0, 64, 0)
	for _, profession := range []VillagerProfession{"", VillagerProfessionNone, VillagerProfessionNitwit} {
		villager.VillagerProfession = profession
		villager.IsBaby = false
		if villager.CanTradeAsVillager() {
			t.Errorf("profession %q was allowed to trade", profession)
		}
	}
	villager.VillagerProfession = VillagerProfessionFarmer
	villager.IsBaby = true
	if villager.CanTradeAsVillager() {
		t.Error("baby villager was allowed to trade")
	}
	villager.IsBaby = false
	if !villager.CanTradeAsVillager() {
		t.Error("adult farmer was not allowed to trade")
	}
}
