package bedrock

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestLevelEntitySlotsMapToOpenChest(t *testing.T) {
	p := player.New([16]byte{42}, "chest", player.ClientEditionBedrock)
	p.OpenContainerKind = "minecraft:chest"
	p.ContainerSlots = make([]player.ItemStack, 27)
	for _, slot := range []byte{0, 13, 26} {
		got, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{
			Container: protocol.FullContainerName{ContainerID: protocol.ContainerLevelEntity},
			Slot:      slot,
		})
		want := intent.InventoryContainerStart + int16(slot)
		if !ok || got != want {
			t.Fatalf("level-entity slot %d = %d/%t, want %d/true", slot, got, ok, want)
		}
	}
	if _, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{
		Container: protocol.FullContainerName{ContainerID: protocol.ContainerLevelEntity}, Slot: 27,
	}); ok {
		t.Fatal("out-of-range chest slot was accepted")
	}
}

func TestBedrockWorkstationSlotsMapToCanonicalContainer(t *testing.T) {
	tests := []struct {
		kind      string
		container byte
		slot      byte
		want      int16
	}{
		{kind: "minecraft:cartography_table", container: protocol.ContainerCartographyInput, want: intent.InventoryContainerStart},
		{kind: "minecraft:cartography_table", container: protocol.ContainerCartographyAdditional, want: intent.InventoryContainerStart + 1},
		{kind: "minecraft:cartography_table", container: protocol.ContainerCartographyResultPreview, want: intent.InventoryContainerStart + 2},
		{kind: "minecraft:smithing_table", container: protocol.ContainerSmithingTableTemplate, want: intent.InventoryContainerStart},
		{kind: "minecraft:smithing_table", container: protocol.ContainerSmithingTableInput, want: intent.InventoryContainerStart + 1},
		{kind: "minecraft:loom", container: protocol.ContainerLoomResultPreview, want: intent.InventoryContainerStart + 3},
		{kind: "minecraft:brewing_stand", container: protocol.ContainerBrewingStandResult, slot: 2, want: intent.InventoryContainerStart + 2},
		{kind: "minecraft:brewing_stand", container: protocol.ContainerBrewingStandFuel, want: intent.InventoryContainerStart + 4},
	}
	for _, test := range tests {
		p := player.New([16]byte{43}, "workstation", player.ClientEditionBedrock)
		p.OpenContainerKind = test.kind
		p.ContainerSlots = make([]player.ItemStack, 5)
		got, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{
			Container: protocol.FullContainerName{ContainerID: test.container}, Slot: test.slot,
		})
		if !ok || got != test.want {
			t.Errorf("%s container %d slot %d = %d/%t, want %d/true", test.kind, test.container, test.slot, got, ok, test.want)
		}
	}
}

func TestAllBedrockWorkstationsHaveInitialSlotDescriptors(t *testing.T) {
	for _, kind := range []string{
		"minecraft:anvil", "minecraft:enchanting_table", "minecraft:grindstone", "minecraft:loom",
		"minecraft:smithing_table", "minecraft:stonecutter", "minecraft:cartography_table",
		"minecraft:brewing_stand", "minecraft:beacon",
	} {
		if slots := bedrockWorkstationSlots(kind); len(slots) == 0 {
			t.Errorf("%s has no Bedrock slot descriptors", kind)
		}
	}
}
