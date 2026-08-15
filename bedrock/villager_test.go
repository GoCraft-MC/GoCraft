package bedrock

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/java/handler"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
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

func TestBedrockVillagerOffersUseNetworkNBT(t *testing.T) {
	offers := handler.VillagerTrades(corentity.VillagerProfessionLibrarian, 3)
	encoded, err := bedrockVillagerOffersNBT(offers)
	if err != nil {
		t.Fatalf("encode offers: %v", err)
	}
	var root struct {
		Recipes             []map[string]any   `nbt:"Recipes"`
		TierExpRequirements []map[string]int32 `nbt:"TierExpRequirements"`
	}
	if err := nbt.Unmarshal(encoded, &root); err != nil {
		t.Fatalf("decode offers: %v", err)
	}
	if len(root.Recipes) != len(offers) || len(root.TierExpRequirements) != 5 {
		t.Fatalf("offer NBT = %d recipes/%v tiers, want %d/5", len(root.Recipes), root.TierExpRequirements, len(offers))
	}
	firstBuy, ok := root.Recipes[0]["buyA"].(map[string]any)
	if !ok || firstBuy["Name"] == "" || root.Recipes[0]["sell"] == nil ||
		root.Recipes[0]["buyCountA"] == nil || root.Recipes[0]["buyCountB"] == nil || root.Recipes[0]["tier"] == nil {
		t.Fatalf("first Bedrock offer is incomplete: %#v", root.Recipes[0])
	}
	if got := firstBuy["Damage"]; got != int16(32767) {
		t.Fatalf("ingredient wildcard damage = %#v, want 32767", got)
	}
	for level, requirement := range []int32{0, 10, 70, 150, 250} {
		if got := root.TierExpRequirements[level][string(rune('0'+level))]; got != requirement {
			t.Fatalf("tier %d requirement = %d, want %d", level, got, requirement)
		}
	}
}

func TestBedrockVillagerTradeTitleIncludesProfessionAndRank(t *testing.T) {
	if got := bedrockVillagerTradeTitle(corentity.VillagerProfessionArmorer, 1); got != "Armorer - Novice" {
		t.Fatalf("trade title = %q", got)
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
	baby := listener.bedrockEntityMetadata(nil, villager)
	if !baby.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBaby) || baby[protocol.EntityDataKeyScale] != float32(0.5) {
		t.Fatalf("baby metadata = %#v", baby)
	}
	villager.IsBaby = false
	adult := listener.bedrockEntityMetadata(nil, villager)
	if adult.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagBaby) || adult[protocol.EntityDataKeyScale] != float32(1) {
		t.Fatalf("adult metadata = %#v", adult)
	}
}

func TestBedrockAnimalMetadataCarriesCanonicalInteractionState(t *testing.T) {
	wolf := corentity.New(20, [16]byte{}, corentity.TypeWolf, 0, 64, 0)
	wolf.IsBaby = true
	wolf.LoveTicks = 100
	wolf.Tamed = true
	wolf.Sitting = true
	wolf.HasTameOwner = true
	wolf.TameOwnerEntityID = 4
	metadata := (&Listener{}).bedrockEntityMetadata(&bedrockSession{entityID: 4}, wolf)
	for _, flag := range []uint8{
		protocol.EntityDataFlagBaby, protocol.EntityDataFlagInLove,
		protocol.EntityDataFlagTamed, protocol.EntityDataFlagSitting,
	} {
		if !metadata.Flag(protocol.EntityDataKeyFlags, flag) {
			t.Fatalf("animal metadata missing flag %d: %#v", flag, metadata)
		}
	}
	if got := metadata[protocol.EntityDataKeyOwner]; got != int64(bedrockSelfRuntimeID) {
		t.Fatalf("owner runtime ID = %#v, want self %d", got, bedrockSelfRuntimeID)
	}

	horse := corentity.New(21, [16]byte{}, corentity.TypeHorse, 0, 64, 0)
	horse.Saddled = true
	if metadata := (&Listener{}).bedrockEntityMetadata(nil, horse); !metadata.Flag(protocol.EntityDataKeyFlags, protocol.EntityDataFlagSaddled) {
		t.Fatal("saddled flag missing from horse metadata")
	}
}
