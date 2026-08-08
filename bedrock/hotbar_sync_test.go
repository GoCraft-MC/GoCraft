package bedrock

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"

	bedrockworld "GoCraft/bedrock/world"
	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func TestRapidClientHotbarSelectionsRemainClientOwned(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{91}, "scrolling-player", player.ClientEditionBedrock)
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	bus := intent.NewBus(1, 32)
	l := &Listener{game: g, bus: bus}
	session := &bedrockSession{inventorySent: true}

	slots := []int32{1, 2, 3, 4, 3, 2, 7, 8}
	for _, slot := range slots {
		if !l.acceptClientHotbarSlot(session, p.UUID, slot, "MobEquipment") {
			t.Fatalf("valid slot %d was rejected", slot)
		}
		if shouldBootstrapHotbarSelection(session.inventorySent, session.clientHeldSlotSeen) {
			t.Fatalf("slot %d incorrectly requested a server selection packet", slot)
		}
	}

	drained := bus.Drain()
	var got []int32
	for _, gameplay := range drained.Gameplay {
		if selection, ok := gameplay.(intent.HotbarIntent); ok {
			got = append(got, selection.Slot)
		}
	}
	if len(got) != len(slots) {
		t.Fatalf("queued selections = %v, want %v", got, slots)
	}
	for index := range slots {
		if got[index] != slots[index] {
			t.Fatalf("queued selections = %v, want FIFO %v", got, slots)
		}
	}
	if session.clientHeldSlot != int(slots[len(slots)-1]) {
		t.Fatalf("latest client slot = %d, want %d", session.clientHeldSlot, slots[len(slots)-1])
	}
}

func TestInventoryMutationDoesNotRebootstrapHotbarSelection(t *testing.T) {
	if !shouldBootstrapHotbarSelection(false, false) {
		t.Fatal("fresh login should bootstrap its persisted selected slot")
	}
	if shouldBootstrapHotbarSelection(false, true) {
		t.Fatal("client selection received during login must not be overwritten")
	}
	if shouldBootstrapHotbarSelection(true, true) {
		t.Fatal("runtime inventory mutation must not resend selected slot")
	}
}

func TestUseItemActionSlotDoesNotBecomeASelectionUpdate(t *testing.T) {
	bus := intent.NewBus(1, 4)
	l := &Listener{bus: bus}
	l.handleUseItemTransaction([16]byte{93}, &protocol.UseItemTransactionData{
		ActionType: protocol.UseItemActionClickBlock,
		HotBarSlot: 6,
	})

	var interactionFound bool
	for _, gameplay := range bus.Drain().Gameplay {
		switch value := gameplay.(type) {
		case intent.HotbarIntent:
			t.Fatalf("item action created a persistent selection update for slot %d", value.Slot)
		case intent.BlockInteractIntent:
			interactionFound = true
			if value.HotbarSlot != 6 {
				t.Fatalf("action slot = %d, want 6", value.HotbarSlot)
			}
		}
	}
	if !interactionFound {
		t.Fatal("valid use-item action was not queued")
	}
}

func TestPlayerActionBreakHasNoImplicitSlotZero(t *testing.T) {
	bus := intent.NewBus(1, 4)
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{92}, "breaking-player", player.ClientEditionBedrock)
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	l := &Listener{bus: bus, game: g, world: w}
	session := &bedrockSession{breaking: true, breakingPos: protocol.BlockPos{1, 64, 1}}
	l.handlePlayerBlockAction(session, p.UUID, protocol.PlayerActionStopBreak, session.breakingPos, 1)

	for _, gameplay := range bus.Drain().Gameplay {
		if interaction, ok := gameplay.(intent.BlockInteractIntent); ok {
			if interaction.HotbarSlot != -1 {
				t.Fatalf("PlayerAction break slot = %d, want -1 (preserve selected slot)", interaction.HotbarSlot)
			}
			return
		}
	}
	t.Fatal("PlayerAction break did not queue a block interaction")
}

func TestBedrockToolDamageIsEncodedForDurabilityBar(t *testing.T) {
	l := &Listener{encoder: bedrockworld.NewEncoder()}
	for _, itemID := range []string{
		"minecraft:wooden_shovel",
		"minecraft:wooden_pickaxe",
		"minecraft:wooden_axe",
		"minecraft:wooden_sword",
		"minecraft:wooden_hoe",
	} {
		t.Run(itemID, func(t *testing.T) {
			instance := l.itemInstance(player.ItemStack{ItemID: itemID, Count: 1, Damage: 1}, 1)
			if got := instance.Stack.NBTData["Damage"]; got != int32(1) {
				t.Fatalf("network Damage = %#v, want int32(1)", got)
			}
		})
	}
}
