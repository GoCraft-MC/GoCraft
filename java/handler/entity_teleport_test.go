package handler

import (
	"bytes"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/protocol"
)

func TestEntityPositionSyncPacketsHaveProtocol769Layout(t *testing.T) {
	t.Run("player", func(t *testing.T) {
		p := player.New([16]byte{1}, "player", player.ClientEditionJava)
		p.EntityID = 42
		p.Position.X, p.Position.Y, p.Position.Z = 1.25, 64, -3.5
		p.Rotation.Yaw, p.Rotation.Pitch = 90, -15
		p.OnGround = true

		assertTeleportEntityPayload(t, buildTeleportEntity(p), 42, true)
	})

	t.Run("mob", func(t *testing.T) {
		e := corentity.New(7, [16]byte{2}, corentity.TypeCow, 4, 70, -8)
		e.VX, e.VY, e.VZ = 0.1, -0.2, 0.3
		e.Yaw, e.Pitch = 45, 10
		e.OnGround = false

		assertTeleportEntityPayload(t, buildTeleportMob(e), 7, false)
	})
}

func assertTeleportEntityPayload(t *testing.T, pkt *protocol.Packet, wantEntityID int32, wantOnGround bool) {
	t.Helper()

	if pkt.ID != packetIDTeleportEntity {
		t.Fatalf("packet ID = %d, want %d", pkt.ID, packetIDTeleportEntity)
	}

	r := pkt.Reader()
	entityID, err := protocol.ReadVarInt(r)
	if err != nil {
		t.Fatalf("read entity ID: %v", err)
	}
	if entityID != wantEntityID {
		t.Fatalf("entity ID = %d, want %d", entityID, wantEntityID)
	}

	for i := 0; i < 6; i++ {
		if _, err := protocol.ReadDouble(r); err != nil {
			t.Fatalf("read double field %d: %v", i, err)
		}
	}
	for i := 0; i < 2; i++ {
		if _, err := protocol.ReadFloat(r); err != nil {
			t.Fatalf("read float field %d: %v", i, err)
		}
	}

	onGround, err := protocol.ReadBool(r)
	if err != nil {
		t.Fatalf("read on-ground flag: %v", err)
	}
	if onGround != wantOnGround {
		t.Fatalf("on-ground = %v, want %v", onGround, wantOnGround)
	}
	if r.Len() != 0 {
		t.Fatalf("unexpected trailing payload: %d bytes", r.Len())
	}
}

func TestVillagerMetadataUsesProtocol769RegistryValues(t *testing.T) {
	villager := corentity.New(77, [16]byte{3}, corentity.TypeVillager, 0, 64, 0)
	villager.VillagerVariant = corentity.VillagerVariantSavanna
	villager.VillagerProfession = corentity.VillagerProfessionLibrarian
	villager.VillagerLevel = 2

	assertVillagerMetadata := func(wantBed *spatial.BlockPos) {
		t.Helper()
		pkt := buildMobMetadata(villager)
		if pkt == nil {
			t.Fatal("buildMobMetadata(villager) = nil")
		}
		r := assertLivingEntitySleepMetadata(t, pkt, villager.EntityID, wantBed)
		index, err := protocol.ReadByte(r)
		if err != nil {
			t.Fatalf("read villager baby index: %v", err)
		}
		if index != 16 {
			t.Fatalf("villager baby index = %d, want 16", index)
		}
		assertMetadataVarInt(t, r, "villager baby serializer ID", 8)
		baby, err := protocol.ReadBool(r)
		if err != nil || baby {
			t.Fatalf("adult villager baby metadata = %v/%v", baby, err)
		}
		index, err = protocol.ReadByte(r)
		if err != nil {
			t.Fatalf("read villager data index: %v", err)
		}
		if index != 18 {
			t.Fatalf("villager data index = %d, want 18", index)
		}
		assertMetadataVarInt(t, r, "villager data serializer ID", 19)
		assertMetadataVarInt(t, r, "savanna variant ID", 3)
		assertMetadataVarInt(t, r, "librarian profession ID", 9)
		assertMetadataVarInt(t, r, "villager level", 2)
		terminator, err := protocol.ReadByte(r)
		if err != nil {
			t.Fatalf("read metadata terminator: %v", err)
		}
		if terminator != 0xff || r.Len() != 0 {
			t.Fatalf("metadata terminator/trailing bytes = 0x%02x/%d, want 0xff/0", terminator, r.Len())
		}
	}

	assertVillagerMetadata(nil)
	villager.Sleeping = true
	villager.VillageBed = spatial.BlockPos{X: -12, Y: 68, Z: 34}
	assertVillagerMetadata(&villager.VillageBed)

	cow := corentity.New(78, [16]byte{4}, corentity.TypeCow, 0, 64, 0)
	if got := buildMobMetadata(cow); got == nil {
		t.Fatal("buildMobMetadata(cow) = nil, want ageable animal metadata")
	}
}

func TestJavaAnimalMetadataSynchronizesBabyTameOwnerAndSaddle(t *testing.T) {
	owner := [16]byte{1, 2, 3, 4}
	wolf := corentity.New(90, [16]byte{9}, corentity.TypeWolf, 0, 64, 0)
	wolf.IsBaby = true
	wolf.Tamed = true
	wolf.Sitting = true
	wolf.HasTameOwner = true
	wolf.TameOwnerUUID = owner
	r := buildMobMetadata(wolf).Reader()
	assertMetadataVarInt(t, r, "wolf entity ID", wolf.EntityID)
	index, _ := protocol.ReadByte(r)
	if index != 16 {
		t.Fatalf("wolf baby index = %d", index)
	}
	assertMetadataVarInt(t, r, "wolf baby serializer", 8)
	baby, _ := protocol.ReadBool(r)
	if !baby {
		t.Fatal("wolf baby metadata is false")
	}
	index, _ = protocol.ReadByte(r)
	if index != 17 {
		t.Fatalf("wolf tame flags index = %d", index)
	}
	assertMetadataVarInt(t, r, "wolf flags serializer", 0)
	flags, _ := protocol.ReadByte(r)
	if flags&0x05 != 0x05 {
		t.Fatalf("wolf flags = %#x, want sitting+tamed", flags)
	}
	index, _ = protocol.ReadByte(r)
	if index != 18 {
		t.Fatalf("wolf owner index = %d", index)
	}
	assertMetadataVarInt(t, r, "wolf owner serializer", 13)
	present, _ := protocol.ReadBool(r)
	gotOwner, err := protocol.ReadUUID(r)
	if !present || err != nil || [16]byte(gotOwner) != owner {
		t.Fatalf("wolf owner = %v/%v/%v", present, gotOwner, err)
	}
	assertMetadataTerminator(t, r)

	// Horse: tamed+saddled — index 16 (baby), 17 (flags), then 0xff only.
	// NO index 18 Optional UUID; that field does not exist for AbstractHorse.
	horse := corentity.New(91, [16]byte{10}, corentity.TypeHorse, 0, 64, 0)
	horse.Tamed, horse.Saddled = true, true
	horse.HasTameOwner = true // must be ignored — AbstractHorse has no owner UUID field
	horse.TameOwnerUUID = [16]byte{0xde, 0xad}
	horseR := buildMobMetadata(horse).Reader()
	assertMetadataVarInt(t, horseR, "horse entity ID", horse.EntityID)
	idx, _ := protocol.ReadByte(horseR)
	if idx != 16 {
		t.Fatalf("horse baby index = %d, want 16", idx)
	}
	assertMetadataVarInt(t, horseR, "horse baby serializer", 8)
	protocol.ReadBool(horseR) // false — skip
	idx, _ = protocol.ReadByte(horseR)
	if idx != 17 {
		t.Fatalf("horse flags index = %d, want 17", idx)
	}
	assertMetadataVarInt(t, horseR, "horse flags serializer", 0)
	horseFlags, _ := protocol.ReadByte(horseR)
	if horseFlags&0x06 != 0x06 {
		t.Fatalf("tamed+saddled horse flags = %#x, want bits 0x02|0x04", horseFlags)
	}
	// Terminator must follow immediately — no index 18 Optional UUID.
	assertMetadataTerminator(t, horseR)
}

func TestJavaPufferfishMetadataCarriesInflationState(t *testing.T) {
	fish := corentity.New(91, [16]byte{}, corentity.TypePufferfish, 0, 64, 0)
	fish.PufferState = 2
	r := buildMobMetadata(fish).Reader()
	assertMetadataVarInt(t, r, "pufferfish entity ID", fish.EntityID)
	index, _ := protocol.ReadByte(r)
	if index != 17 {
		t.Fatalf("puff state index = %d, want 17", index)
	}
	assertMetadataVarInt(t, r, "puff state serializer", 1)
	assertMetadataVarInt(t, r, "puff state", 2)
}

// TestJavaAbstractHorseMetadataHasNoOwnerUUIDAtIndex18 is a protocol-level
// regression test for the bug introduced in commit 7c3ec145: writing serializer
// type 13 (Optional UUID) at metadata index 18 for every AbstractHorse entity.
// ZombieHorse can spawn naturally, so every joining Java 1.21.4 client would
// receive the malformed packet and disconnect with "Network Protocol Error".
func TestJavaAbstractHorseMetadataHasNoOwnerUUIDAtIndex18(t *testing.T) {
	horseTypes := []corentity.EntityType{
		corentity.TypeHorse,
		corentity.TypeDonkey,
		corentity.TypeMule,
		corentity.TypeSkeletonHorse,
		corentity.TypeZombieHorse,
		corentity.TypeCamel,
		corentity.TypeLlama,
		corentity.TypeTraderLlama,
	}
	for i, ht := range horseTypes {
		t.Run(string(ht), func(t *testing.T) {
			e := corentity.New(int32(300+i), [16]byte{byte(i + 1)}, ht, 0, 64, 0)
			e.HasTameOwner = true // worst case: would emit UUID if bug were present
			e.TameOwnerUUID = [16]byte{0xde, 0xad, 0xbe, 0xef}
			pkt := buildMobMetadata(e)
			if pkt == nil {
				t.Fatal("AbstractHorse metadata is nil")
			}
			r := pkt.Reader()
			assertMetadataVarInt(t, r, "entity ID", e.EntityID)
			// index 16: baby (from AgeableMob — all AbstractHorse subtypes are ageable)
			idx, err := protocol.ReadByte(r)
			if err != nil || idx != 16 {
				t.Fatalf("baby index = %d err = %v, want 16", idx, err)
			}
			assertMetadataVarInt(t, r, "baby serializer", 8)
			protocol.ReadBool(r) // false — skip
			// index 17: horse flags byte
			idx, err = protocol.ReadByte(r)
			if err != nil || idx != 17 {
				t.Fatalf("flags index = %d err = %v, want 17", idx, err)
			}
			assertMetadataVarInt(t, r, "flags serializer", 0)
			protocol.ReadByte(r) // flags value — skip
			// Next byte MUST be the terminator 0xff.
			// If it is 18, the index-18 Optional UUID bug is present.
			b, err := protocol.ReadByte(r)
			if err != nil {
				t.Fatalf("read after flags: %v", err)
			}
			if b == 18 {
				t.Fatalf("%s metadata contains index 18 — Optional UUID bug is present (causes Java client disconnect)", ht)
			}
			if b != 0xff {
				t.Fatalf("%s metadata has unexpected byte 0x%02x after flags (want 0xff terminator)", ht, b)
			}
			if r.Len() != 0 {
				t.Fatalf("%s metadata has %d trailing bytes", ht, r.Len())
			}
		})
	}
}

// TestJavaUntamedHorseFlagsAreZero verifies default (untamed, unsaddled) horse
// writes flags=0x00 at index 17 and terminates immediately.
func TestJavaUntamedHorseFlagsAreZero(t *testing.T) {
	horse := corentity.New(400, [16]byte{20}, corentity.TypeHorse, 0, 64, 0)
	r := buildMobMetadata(horse).Reader()
	assertMetadataVarInt(t, r, "entity ID", horse.EntityID)
	idx, _ := protocol.ReadByte(r)
	if idx != 16 {
		t.Fatalf("baby index = %d", idx)
	}
	assertMetadataVarInt(t, r, "baby serializer", 8)
	protocol.ReadBool(r)
	idx, _ = protocol.ReadByte(r)
	if idx != 17 {
		t.Fatalf("flags index = %d", idx)
	}
	assertMetadataVarInt(t, r, "flags serializer", 0)
	flags, _ := protocol.ReadByte(r)
	if flags != 0 {
		t.Fatalf("untamed horse flags = %#x, want 0x00", flags)
	}
	assertMetadataTerminator(t, r)
}

func TestPlayerSleepMetadataUsesProtocol769PoseAndBedPosition(t *testing.T) {
	bed := spatial.BlockPos{X: 123, Y: 70, Z: -456}

	sleeping := buildPlayerPoseMetadata(42, true, bed)
	r := assertLivingEntitySleepMetadata(t, sleeping, 42, &bed)
	assertMetadataTerminator(t, r)

	waking := buildPlayerPoseMetadata(42, false, spatial.BlockPos{})
	r = assertLivingEntitySleepMetadata(t, waking, 42, nil)
	assertMetadataTerminator(t, r)
}

func assertLivingEntitySleepMetadata(t *testing.T, pkt *protocol.Packet, wantEntityID int32, wantBed *spatial.BlockPos) *bytes.Reader {
	t.Helper()
	if pkt.ID != packetIDSetEntityData {
		t.Fatalf("packet ID = %d, want %d", pkt.ID, packetIDSetEntityData)
	}
	r := pkt.Reader()
	assertMetadataVarInt(t, r, "entity ID", wantEntityID)

	index, err := protocol.ReadByte(r)
	if err != nil {
		t.Fatalf("read pose index: %v", err)
	}
	if index != entityMetadataPoseIndex {
		t.Fatalf("pose index = %d, want %d", index, entityMetadataPoseIndex)
	}
	assertMetadataVarInt(t, r, "pose serializer ID", metadataTypePose)
	wantPose := entityPoseStanding
	if wantBed != nil {
		wantPose = entityPoseSleeping
	}
	assertMetadataVarInt(t, r, "pose", wantPose)

	index, err = protocol.ReadByte(r)
	if err != nil {
		t.Fatalf("read sleeping position index: %v", err)
	}
	if index != livingEntityMetadataSleepingPosIndex {
		t.Fatalf("sleeping position index = %d, want %d", index, livingEntityMetadataSleepingPosIndex)
	}
	assertMetadataVarInt(t, r, "sleeping position serializer ID", metadataTypeOptionalBlockPos)
	present, err := protocol.ReadBool(r)
	if err != nil {
		t.Fatalf("read sleeping position presence: %v", err)
	}
	if present != (wantBed != nil) {
		t.Fatalf("sleeping position presence = %v, want %v", present, wantBed != nil)
	}
	if wantBed != nil {
		packed, err := protocol.ReadLong(r)
		if err != nil {
			t.Fatalf("read sleeping position: %v", err)
		}
		if got := spatial.DecodeBlockPos(packed); got != *wantBed {
			t.Fatalf("sleeping position = %v, want %v", got, *wantBed)
		}
	}
	return r
}

func assertMetadataVarInt(t *testing.T, r *bytes.Reader, name string, want int32) {
	t.Helper()
	got, err := protocol.ReadVarInt(r)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", name, got, want)
	}
}

func assertMetadataTerminator(t *testing.T, r *bytes.Reader) {
	t.Helper()
	terminator, err := protocol.ReadByte(r)
	if err != nil {
		t.Fatalf("read metadata terminator: %v", err)
	}
	if terminator != 0xff || r.Len() != 0 {
		t.Fatalf("metadata terminator/trailing bytes = 0x%02x/%d, want 0xff/0", terminator, r.Len())
	}
}
