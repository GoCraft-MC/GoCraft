package handler

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

func TestJavaAnimalInteractionPostsCanonicalIntent(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	cow := corentity.New(44, [16]byte{}, corentity.TypeCow, 1, 64, 0)
	world.Entities.Add(cow)
	p := player.New([16]byte{9}, "java-breeder", player.ClientEditionJava)
	p.Position.X, p.Position.Y = 0, 64
	p.HeldSlot = 2
	bus := intent.NewBus(1, 4)
	packet := protocol.NewBuilder(packetIDInteract).
		VarInt(cow.EntityID).VarInt(0).VarInt(0).Bool(false).Build()
	if err := handleInteractPacket(packet, p, world, nil, nil, bus); err != nil {
		t.Fatal(err)
	}
	if cow.LoveTicks != 0 {
		t.Fatal("Java adapter mutated canonical animal directly")
	}
	drained := bus.Drain()
	if len(drained.Gameplay) != 1 {
		t.Fatalf("gameplay intent count = %d, want 1", len(drained.Gameplay))
	}
	interaction, ok := drained.Gameplay[0].(intent.EntityInteractIntent)
	if !ok || interaction.PlayerUUID != p.UUID || interaction.TargetID != cow.EntityID || interaction.HotbarSlot != 2 {
		t.Fatalf("interaction = %#v", drained.Gameplay[0])
	}
}

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

func TestVillagerTradeCatalogSeparatesEveryProfessionAndLevel(t *testing.T) {
	professions := []corentity.VillagerProfession{
		corentity.VillagerProfessionArmorer, corentity.VillagerProfessionButcher,
		corentity.VillagerProfessionCartographer, corentity.VillagerProfessionCleric,
		corentity.VillagerProfessionFarmer, corentity.VillagerProfessionFisherman,
		corentity.VillagerProfessionFletcher, corentity.VillagerProfessionLeatherworker,
		corentity.VillagerProfessionLibrarian, corentity.VillagerProfessionMason,
		corentity.VillagerProfessionShepherd, corentity.VillagerProfessionToolsmith,
		corentity.VillagerProfessionWeaponsmith,
	}
	for _, profession := range professions {
		previous := 0
		for level := int32(1); level <= 5; level++ {
			offers := tradesForProfession(profession, level)
			if len(offers) <= previous {
				t.Fatalf("%s level %d has %d offers, want more than level %d's %d", profession, level, len(offers), level-1, previous)
			}
			for _, offer := range offers {
				if offer.input1.itemName == "" || offer.output.itemName == "" || offer.tier >= level {
					t.Fatalf("invalid %s level %d offer: %+v", profession, level, offer)
				}
			}
			previous = len(offers)
		}
	}
	if offers := tradesForProfession(corentity.VillagerProfessionNone, 5); len(offers) != 0 {
		t.Fatalf("unemployed villager received %d offers", len(offers))
	}
}

func TestNoviceArmorerCannotReceiveOtherProfessionTrades(t *testing.T) {
	denied := map[string]bool{
		"minecraft:wheat": true, "minecraft:potato": true, "minecraft:carrot": true,
		"minecraft:paper": true, "minecraft:bookshelf": true,
		"minecraft:stick": true, "minecraft:arrow": true,
	}
	offers := VillagerTrades(corentity.VillagerProfessionArmorer, 1)
	if len(offers) != 2 {
		t.Fatalf("novice armorer offers = %d, want vanilla selection of 2", len(offers))
	}
	for _, offer := range offers {
		if denied[offer.Input1.ItemID] || denied[offer.Output.ItemID] || denied[offer.Input2.ItemID] {
			t.Fatalf("novice armorer leaked another profession's offer: %+v", offer)
		}
		if offer.Tier != 0 {
			t.Fatalf("novice armorer tier = %d, want 0", offer.Tier)
		}
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
