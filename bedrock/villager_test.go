package bedrock

import (
	"testing"

	corentity "GoCraft/core/entity"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestBedrockVillagerSpawnCarriesProfessionAndBiomeVariant(t *testing.T) {
	villager := corentity.New(7, [16]byte{}, corentity.TypeVillager, 1, 64, 1)
	villager.VillagerProfession = corentity.VillagerProfessionLibrarian
	villager.VillagerVariant = corentity.VillagerVariantDesert
	villager.VillagerLevel = 3

	spawn, ok := (&Listener{}).buildAddEntity(&bedrockSession{}, villager).(*packet.AddActor)
	if !ok {
		t.Fatal("villager did not build an AddActor packet")
	}
	if spawn.EntityType != "minecraft:villager_v2" {
		t.Fatalf("Bedrock villager type = %q, want minecraft:villager_v2", spawn.EntityType)
	}
	if got := spawn.EntityMetadata[protocol.EntityDataKeyVariant]; got != int32(5) {
		t.Fatalf("librarian profession variant = %#v, want 5", got)
	}
	if got := spawn.EntityMetadata[protocol.EntityDataKeyMarkVariant]; got != int32(1) {
		t.Fatalf("desert mark variant = %#v, want 1", got)
	}
	if got := spawn.EntityMetadata[protocol.EntityDataKeyTradeTier]; got != int32(2) {
		t.Fatalf("trade tier = %#v, want zero-based tier 2", got)
	}
}

func TestBedrockVillagerProfessionIDsCoverEveryCanonicalJob(t *testing.T) {
	want := map[corentity.VillagerProfession]int32{
		corentity.VillagerProfessionNone:          0,
		corentity.VillagerProfessionFarmer:        1,
		corentity.VillagerProfessionFisherman:     2,
		corentity.VillagerProfessionShepherd:      3,
		corentity.VillagerProfessionFletcher:      4,
		corentity.VillagerProfessionLibrarian:     5,
		corentity.VillagerProfessionCartographer:  6,
		corentity.VillagerProfessionCleric:        7,
		corentity.VillagerProfessionArmorer:       8,
		corentity.VillagerProfessionWeaponsmith:   9,
		corentity.VillagerProfessionToolsmith:     10,
		corentity.VillagerProfessionButcher:       11,
		corentity.VillagerProfessionLeatherworker: 12,
		corentity.VillagerProfessionMason:         13,
		corentity.VillagerProfessionNitwit:        14,
	}
	for profession, id := range want {
		if got := bedrockVillagerProfessionID(profession); got != id {
			t.Errorf("profession %s ID = %d, want %d", profession, got, id)
		}
	}
}

func TestBedrockBabyVillagerMetadataCanReturnToAdultScale(t *testing.T) {
	villager := corentity.New(8, [16]byte{}, corentity.TypeVillager, 1, 64, 1)
	villager.IsBaby = true
	listener := &Listener{}
	baby := listener.bedrockEntityMetadata(villager)
	if !baby.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBaby) || baby[protocol.EntityDataKeyScale] != float32(0.5) {
		t.Fatalf("baby metadata = %#v", baby)
	}
	villager.IsBaby = false
	adult := listener.bedrockEntityMetadata(villager)
	if adult.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBaby) || adult[protocol.EntityDataKeyScale] != float32(1) {
		t.Fatalf("adult metadata = %#v", adult)
	}
}
