package entity

import "testing"

func TestAnimalFoodTables(t *testing.T) {
	tests := []struct {
		name   string
		mob    EntityType
		item   string
		tamed  bool
		breed  bool
		taming bool
	}{
		{"cow wheat", TypeCow, "minecraft:wheat", false, true, false},
		{"chicken pitcher pod", TypeChicken, "minecraft:pitcher_pod", false, true, false},
		{"pig carrot", TypePig, "minecraft:carrot", false, true, false},
		{"pig carrot stick rejected", TypePig, "minecraft:carrot_on_a_stick", false, false, false},
		{"wolf bone", TypeWolf, "minecraft:bone", false, false, true},
		{"owned wolf meat", TypeWolf, "minecraft:cooked_beef", true, true, false},
		{"cat fish tame", TypeCat, "minecraft:salmon", false, false, true},
		{"parrot seeds tame", TypeParrot, "minecraft:wheat_seeds", false, false, true},
		{"horse golden carrot before tame", TypeHorse, "minecraft:golden_carrot", false, false, false},
		{"horse golden carrot after tame", TypeHorse, "minecraft:golden_carrot", true, true, false},
		{"camel cactus", TypeCamel, "minecraft:cactus", false, true, false},
		{"armadillo spider eye", TypeArmadillo, "minecraft:spider_eye", false, true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effect := FoodEffect(test.mob, test.item, test.tamed)
			if effect.Breeding != test.breed || effect.Taming != test.taming {
				t.Fatalf("effect = %+v, want breeding=%v taming=%v", effect, test.breed, test.taming)
			}
			wantAccepted := test.breed || test.taming || test.name == "horse golden carrot before tame"
			if test.name == "pig carrot stick rejected" {
				wantAccepted = false
			}
			if effect.Accepted != wantAccepted {
				t.Fatalf("accepted = %v, want %v", effect.Accepted, wantAccepted)
			}
		})
	}
}

func TestBreedingCompatibilityAndVehicleSeats(t *testing.T) {
	if !CompatibleBreedingTypes(TypeHorse, TypeDonkey) || BreedingChildType(TypeHorse, TypeDonkey) != TypeMule {
		t.Fatal("horse/donkey cross must produce a mule")
	}
	if CompatibleBreedingTypes(TypeMule, TypeMule) {
		t.Fatal("mules must be sterile")
	}
	camel := New(1, [16]byte{}, TypeCamel, 0, 0, 0)
	if !camel.AddPassenger(10) || !camel.AddPassenger(11) || camel.AddPassenger(12) {
		t.Fatalf("camel passengers = %v, want exactly two seats", camel.PassengerIDs())
	}
	removed := camel.RemovePassenger(10)
	got := camel.PassengerIDs()
	if !removed || len(got) != 1 || got[0] != 11 {
		t.Fatalf("passengers after front dismount = %v, want [11]", got)
	}
	if !IsMinecart(TypeMinecart) || !IsRideableVehicle(TypeMinecart) || VehicleCapacity(TypeMinecart) != 1 {
		t.Fatal("minecart must expose one rideable seat")
	}
}
