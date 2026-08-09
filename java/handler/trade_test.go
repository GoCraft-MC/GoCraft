package handler

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

func TestInteractAttackQueuesDamageForTickThread(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	mob := corentity.New(42, [16]byte{}, corentity.TypeCow, 0, 64, 0)
	world.Entities.Add(mob)
	attacker := player.New([16]byte{}, "attacker", player.ClientEditionJava)
	attacker.GameMode = player.GameModeSurvival

	packet := protocol.NewBuilder(packetIDInteract).
		VarInt(mob.EntityID).
		VarInt(1).
		Bool(false).
		Build()
	if err := handleInteractPacket(packet, attacker, world, nil, nil); err != nil {
		t.Fatal(err)
	}
	if mob.Health != mob.MaxHealth {
		t.Fatalf("handler mutated health directly: got %.1f, want %.1f", mob.Health, mob.MaxHealth)
	}
	pending := world.DrainEntityDamage()
	if got := pending[mob.EntityID]; got.Amount != 1 || !got.HasSource {
		t.Fatalf("queued damage = %+v, want amount 1 with attacker source", got)
	}
}

func TestJavaAttackDamagesExternalBedrockPlayer(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	attacker := player.New([16]byte{1}, "java", player.ClientEditionJava)
	target := player.New([16]byte{2}, "bedrock", player.ClientEditionBedrock)
	attacker.GameMode = player.GameModeSurvival
	target.GameMode = player.GameModeSurvival
	attacker.Position.X, target.Position.X = 0, 1
	manager := session.NewManager()
	manager.ReplaceExternalPlayers([]*player.Player{target})
	packet := protocol.NewBuilder(packetIDInteract).VarInt(target.EntityID).VarInt(1).Bool(false).Build()
	if err := handleInteractPacket(packet, attacker, world, nil, manager); err != nil {
		t.Fatal(err)
	}
	health, _, _, _ := target.HealthSnapshot()
	if health >= target.MaxHealth {
		t.Fatalf("Bedrock target health = %v, want Java damage", health)
	}
}

func TestBasicVillagerInteractionUsesUnhappyPathWithoutMerchantScreen(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	villager := corentity.New(43, [16]byte{}, corentity.TypeVillager, 0, 64, 0)
	villager.VillagerProfession = corentity.VillagerProfessionNone
	world.Entities.Add(villager)
	p := player.New([16]byte{3}, "visitor", player.ClientEditionJava)
	packet := protocol.NewBuilder(packetIDInteract).
		VarInt(villager.EntityID).
		VarInt(0).
		VarInt(0).
		Bool(false).
		Build()
	// A nil connection is intentional: the unhappy path must return before the
	// merchant-screen writer is reached.
	if err := handleInteractPacket(packet, p, world, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestMerchantOfferUsesProtocol769ItemCostLayout(t *testing.T) {
	trade := tradeOffer{
		input1:  tradeItem{"minecraft:wheat", 20},
		output:  tradeItem{"minecraft:emerald", 1},
		maxUses: 12,
	}
	packet := buildMerchantOffers(7, []tradeOffer{trade})
	if packet.ID != 0x2e {
		t.Fatalf("merchant packet ID = %#x, want 0x2e", packet.ID)
	}
	r := packet.Reader()
	windowID, _ := protocol.ReadVarInt(r)
	offerCount, _ := protocol.ReadVarInt(r)
	costID, _ := protocol.ReadVarInt(r)
	costCount, _ := protocol.ReadVarInt(r)
	requiredComponents, _ := protocol.ReadVarInt(r)
	outputCount, _ := protocol.ReadVarInt(r)
	outputID, _ := protocol.ReadVarInt(r)
	addedComponents, _ := protocol.ReadVarInt(r)
	removedComponents, _ := protocol.ReadVarInt(r)

	if windowID != 7 || offerCount != 1 {
		t.Fatalf("merchant header = (%d,%d), want (7,1)", windowID, offerCount)
	}
	if costID != javaworld.ItemID("minecraft:wheat") || costCount != 20 || requiredComponents != 0 {
		t.Fatalf("input ItemCost = (%d,%d,%d), want wheat/20/no components", costID, costCount, requiredComponents)
	}
	if outputCount != 1 || outputID != javaworld.ItemID("minecraft:emerald") || addedComponents != 0 || removedComponents != 0 {
		t.Fatalf("output Slot = (%d,%d,%d,%d), want one emerald without components", outputCount, outputID, addedComponents, removedComponents)
	}
}

func TestVillagerUnhappyFeedbackContainsParticlesAndNoSound(t *testing.T) {
	villager := corentity.New(91, [16]byte{}, corentity.TypeVillager, 0, 64, 0)
	packets := villagerUnhappyPackets(villager)
	if len(packets) != 2 {
		t.Fatalf("unhappy feedback packet count = %d, want 2", len(packets))
	}
	if packets[0].ID != packetIDEntityEvent {
		t.Fatalf("first unhappy packet ID = %#x, want entity event %#x", packets[0].ID, packetIDEntityEvent)
	}
	if packets[1].ID != packetIDSoundEntity {
		t.Fatalf("second unhappy packet ID = %#x, want entity sound %#x", packets[1].ID, packetIDSoundEntity)
	}
}
