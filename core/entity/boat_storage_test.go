package entity

import "testing"

func TestChestBoatsHaveStorageAndOneSeat(t *testing.T) {
	for _, kind := range []EntityType{TypeOakChestBoat, TypeSpruceChestBoat, TypeBirchChestBoat,
		TypeJungleChestBoat, TypeAcaciaChestBoat, TypeDarkOakChestBoat,
		TypeMangroveChestBoat, TypeCherryChestBoat, TypeBambooChestRaft} {
		boat := New(42, [16]byte{}, kind, 0, 64, 0)
		if !IsChestBoat(kind) || len(boat.Storage.Snapshot()) != 27 || !boat.AddPassenger(1) || boat.AddPassenger(2) {
			t.Errorf("%s: slots=%d passengers=%v", kind, len(boat.Storage.Snapshot()), boat.PassengerIDs())
		}
	}
	boat := New(43, [16]byte{}, TypeOakBoat, 0, 64, 0)
	if IsChestBoat(boat.Type) || boat.Storage != nil || !boat.AddPassenger(1) || !boat.AddPassenger(2) {
		t.Fatal("ordinary boat must have two seats and no storage")
	}
}
