package handler

import (
	"strings"
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

func TestCreativeInventoryUsesProtocol769ItemIDs(t *testing.T) {
	tests := []struct {
		itemID int32
		name   string
	}{
		{40, "minecraft:acacia_planks"},
		{195, "minecraft:glass"},
		{314, "minecraft:crafting_table"},
		{316, "minecraft:furnace"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := player.New([16]byte{}, "builder", player.ClientEditionJava)
			pkt := protocol.NewBuilder(packetIDCreativeModeSetItem).
				Short(player.HotbarStart).
				VarInt(64).
				VarInt(tc.itemID).
				VarInt(0).
				VarInt(0).
				Build()
			if err := handleCreativeModeSetItem(pkt, p); err != nil {
				t.Fatalf("handleCreativeModeSetItem: %v", err)
			}
			got := p.Inventory[player.HotbarStart]
			if got.ItemID != tc.name || got.Count != 64 {
				t.Fatalf("hotbar item = %+v, want ItemID=%q Count=64", got, tc.name)
			}
		})
	}
}

func TestJavaBreakingSupportRemovesFloatingGrassImmediately(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(0, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "dirt"})
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "short_grass"})
	applyBlockChange(0, 63, 0, coreworld.Air, w, mgr)
	breakUnsupportedBlocksAbove(0, 63, 0, w, mgr)
	if got := w.GetBlock(0, 64, 0); !got.IsAir() {
		t.Fatalf("grass above broken support = %q, want air", got.ResourceLocation())
	}
}

func TestUseItemOnProtocol769LayoutPlacesExactBlock(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:acacia_planks", Count: 64}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	mgr := session.NewManager()

	pkt := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).
		Long(packBlockPos(0, 63, 0)).
		VarInt(1).
		Float(0.5).
		Float(1.0).
		Float(0.5).
		Bool(false).
		Bool(false).
		VarInt(300).
		Build()
	if err := handleUseItemOn(pkt, p, w, mgr, nil, nil); err != nil {
		t.Fatalf("handleUseItemOn: %v", err)
	}
	got := w.GetBlock(0, 64, 0)
	if got.ResourceLocation() != "minecraft:acacia_planks" {
		t.Fatalf("placed block = %q, want minecraft:acacia_planks", got.ResourceLocation())
	}
}

func TestUseItemOnRequiresSequenceAfterWorldBorderHit(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	mgr := session.NewManager()

	pkt := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).
		Long(packBlockPos(0, 63, 0)).
		VarInt(1).
		Float(0.5).
		Float(1.0).
		Float(0.5).
		Bool(false).
		Bool(false).
		Build()
	err := handleUseItemOn(pkt, p, w, mgr, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("handleUseItemOn error = %v, want missing sequence error after world_border_hit", err)
	}
}

func TestBreakingOneGlassBlockDoesNotBreakAnother(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	mgr := session.NewManager()
	glass := coreworld.Block{Namespace: "minecraft", Name: "glass"}
	w.SetBlock(1, 64, 0, glass)
	w.SetBlock(2, 64, 0, glass)

	pkt := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusStartDigging).
		Long(packBlockPos(1, 64, 0)).
		Byte(1).
		VarInt(301).
		Build()
	if err := handlePlayerAction(pkt, p, w, mgr); err != nil {
		t.Fatalf("handlePlayerAction: %v", err)
	}
	if got := w.GetBlock(1, 64, 0); !got.IsAir() {
		t.Fatalf("target block = %q, want air", got.ResourceLocation())
	}
	if got := w.GetBlock(2, 64, 0); !got.Equal(glass) {
		t.Fatalf("neighbor block = %q, want glass", got.ResourceLocation())
	}
}

func TestUseItemOnTogglesBothDoorHalves(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	mgr := session.NewManager()
	lower := coreworld.Block{
		Namespace: "minecraft",
		Name:      "acacia_door",
		Properties: map[string]string{
			"facing": "south", "half": "lower", "hinge": "left",
			"open": "false", "powered": "false",
		},
	}
	upper := copyBlockProperties(lower)
	upper.Properties["half"] = "upper"
	w.SetBlock(0, 64, 0, lower)
	w.SetBlock(0, 65, 0, upper)

	pkt := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).
		Long(packBlockPos(0, 64, 0)).
		VarInt(3).
		Float(0.5).
		Float(0.5).
		Float(0.5).
		Bool(false).
		Bool(false).
		VarInt(302).
		Build()
	if err := handleUseItemOn(pkt, p, w, mgr, nil, nil); err != nil {
		t.Fatalf("handleUseItemOn: %v", err)
	}
	for y := 64; y <= 65; y++ {
		if got := w.GetBlock(0, y, 0).Properties["open"]; got != "true" {
			t.Fatalf("door half at y=%d open=%q, want true", y, got)
		}
	}
}

func TestSurvivalBreaksOnlyOnFinishDigging(t *testing.T) {
	p := player.New([16]byte{}, "survivor", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(3, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "dirt"})

	start := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusStartDigging).
		Long(packBlockPos(3, 64, 0)).
		Byte(1).
		VarInt(303).
		Build()
	if err := handlePlayerAction(start, p, w, mgr); err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(3, 64, 0); got.IsAir() {
		t.Fatal("survival START_DIGGING broke the block before mining finished")
	}

	finish := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusFinishDigging).
		Long(packBlockPos(3, 64, 0)).
		Byte(1).
		VarInt(304).
		Build()
	if err := handlePlayerAction(finish, p, w, mgr); err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(3, 64, 0); !got.IsAir() {
		t.Fatalf("survival target = %q, want air after FINISH_DIGGING", got.ResourceLocation())
	}
	if got := p.Inventory[player.HotbarStart]; got.ItemID != "minecraft:dirt" || got.Count != 1 {
		t.Fatalf("survival drop = %+v, want one dirt", got)
	}
}

func TestBreakingLowerDoublePlantRemovesUpperHalfAndDropsFlower(t *testing.T) {
	p := player.New([16]byte{}, "gardener", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	lower := coreworld.Block{Namespace: "minecraft", Name: "peony", Properties: map[string]string{"half": "lower"}}
	upper := coreworld.Block{Namespace: "minecraft", Name: "peony", Properties: map[string]string{"half": "upper"}}
	w.SetBlock(5, 64, 0, lower)
	w.SetBlock(5, 65, 0, upper)

	start := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusStartDigging).
		Long(packBlockPos(5, 64, 0)).
		Byte(1).
		VarInt(305).
		Build()
	if err := handlePlayerAction(start, p, w, mgr); err != nil {
		t.Fatal(err)
	}
	for y := 64; y <= 65; y++ {
		if got := w.GetBlock(5, y, 0); !got.IsAir() {
			t.Fatalf("plant half y=%d = %q, want air", y, got.ResourceLocation())
		}
	}
	if got := p.Inventory[player.HotbarStart]; got.ItemID != "minecraft:peony" || got.Count != 1 {
		t.Fatalf("flower drop = %+v, want one peony", got)
	}
}

func TestBreakingUpperDoublePlantRemovesLowerHalf(t *testing.T) {
	p := player.New([16]byte{}, "gardener", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	lower := coreworld.Block{Namespace: "minecraft", Name: "lilac", Properties: map[string]string{"half": "lower"}}
	upper := coreworld.Block{Namespace: "minecraft", Name: "lilac", Properties: map[string]string{"half": "upper"}}
	w.SetBlock(5, 64, 0, lower)
	w.SetBlock(5, 65, 0, upper)

	start := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusStartDigging).
		Long(packBlockPos(5, 65, 0)).
		Byte(1).
		VarInt(306).
		Build()
	if err := handlePlayerAction(start, p, w, mgr); err != nil {
		t.Fatal(err)
	}
	for y := 64; y <= 65; y++ {
		if got := w.GetBlock(5, y, 0); !got.IsAir() {
			t.Fatalf("plant half y=%d = %q, want air", y, got.ResourceLocation())
		}
	}
}

func TestSurvivalGrassBreaksOnStartDigging(t *testing.T) {
	p := player.New([16]byte{}, "gardener", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(6, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "short_grass"})

	start := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusStartDigging).
		Long(packBlockPos(6, 64, 0)).
		Byte(1).
		VarInt(306).
		Build()
	if err := handlePlayerAction(start, p, w, mgr); err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(6, 64, 0); !got.IsAir() {
		t.Fatalf("grass = %q, want air after START_DIGGING", got.ResourceLocation())
	}
	// Vanilla 1.21.4 gives short grass a 12.5% seed chance. Either result is
	// valid; this test is about instant breaking rather than forcing a drop.
	if got := p.Inventory[player.HotbarStart]; !got.IsEmpty() && (got.ItemID != "minecraft:wheat_seeds" || got.Count != 1) {
		t.Fatalf("grass drop = %+v, want empty or one wheat seed", got)
	}
}

func TestHoeTillsAndSeedsPlant(t *testing.T) {
	p := player.New([16]byte{}, "farmer", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:iron_hoe", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(8, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "dirt"})

	till := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).Long(packBlockPos(8, 64, 0)).VarInt(1).
		Float(0.5).Float(1).Float(0.5).Bool(false).Bool(false).VarInt(400).Build()
	if err := handleUseItemOn(till, p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	farmland := w.GetBlock(8, 64, 0)
	if farmland.ResourceLocation() != "minecraft:farmland" || farmland.Properties["moisture"] != "0" {
		t.Fatalf("tilled block = %s, want dry farmland", farmland.Key())
	}

	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:wheat_seeds", Count: 2}
	plant := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).Long(packBlockPos(8, 64, 0)).VarInt(1).
		Float(0.5).Float(1).Float(0.5).Bool(false).Bool(false).VarInt(401).Build()
	if err := handleUseItemOn(plant, p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	crop := w.GetBlock(8, 65, 0)
	if crop.ResourceLocation() != "minecraft:wheat" || crop.Properties["age"] != "0" {
		t.Fatalf("planted crop = %s, want age-0 wheat", crop.Key())
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 1 {
		t.Fatalf("seed count = %d, want 1", got)
	}
}

func useItemOnPacket(x, y, z int, face, sequence int32) *protocol.Packet {
	return protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).Long(packBlockPos(x, y, z)).VarInt(face).
		Float(0.5).Float(0.5).Float(0.5).Bool(false).Bool(false).VarInt(sequence).Build()
}

func TestJavaDoorPlacementCreatesAndBreaksBothHalves(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:spruce_door", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(12, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	if err := handleUseItemOn(useItemOnPacket(12, 63, 0, 1, 500), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	lower, upper := w.GetBlock(12, 64, 0), w.GetBlock(12, 65, 0)
	if lower.ResourceLocation() != "minecraft:spruce_door" || lower.Properties["half"] != "lower" ||
		upper.ResourceLocation() != "minecraft:spruce_door" || upper.Properties["half"] != "upper" {
		t.Fatalf("door halves = %s / %s", lower.Key(), upper.Key())
	}
	breakLinkedDoorHalf(12, 64, 0, lower, w, mgr)
	if !w.GetBlock(12, 65, 0).IsAir() {
		t.Fatal("breaking the lower door left the upper half")
	}
}

func TestJavaSneakingPlacesAgainstContainer(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	p.Sneaking = true
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:stone", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(14, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "barrel"})

	if err := handleUseItemOn(useItemOnPacket(14, 64, 0, 1, 501), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(14, 65, 0).ResourceLocation(); got != "minecraft:stone" {
		t.Fatalf("block above barrel = %q, want stone", got)
	}
}

func TestJavaPlacesRedstoneWireFromDustItem(t *testing.T) {
	p := player.New([16]byte{}, "engineer", player.ClientEditionJava)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:redstone", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(16, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	if err := handleUseItemOn(useItemOnPacket(16, 63, 0, 1, 502), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	wire := w.GetBlock(16, 64, 0)
	if wire.ResourceLocation() != "minecraft:redstone_wire" || wire.Properties["power"] != "0" {
		t.Fatalf("placed wire = %s", wire.Key())
	}
}

func TestJavaBucketPickupAndPlacement(t *testing.T) {
	p := player.New([16]byte{}, "plumber", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:bucket", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(18, 64, 0, coreworld.MakeFluid("minecraft:water", 0))

	if err := handleUseItemOn(useItemOnPacket(18, 64, 0, 1, 503), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	if !w.GetBlock(18, 64, 0).IsAir() || p.HeldItem().ItemID != "minecraft:water_bucket" {
		t.Fatalf("pickup result block=%s held=%+v", w.GetBlock(18, 64, 0).Key(), p.HeldItem())
	}
	w.SetBlock(18, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	if err := handleUseItemOn(useItemOnPacket(18, 63, 0, 1, 504), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(18, 64, 0); got.ResourceLocation() != "minecraft:water" || coreworld.FluidLevel(got) != 0 {
		t.Fatalf("placed fluid = %s", got.Key())
	}
	if p.HeldItem().ItemID != "minecraft:bucket" {
		t.Fatalf("held after placement = %+v", p.HeldItem())
	}
}

func TestJavaDecoratedPotStoresAnItem(t *testing.T) {
	p := player.New([16]byte{}, "potter", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:diamond", Count: 2}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(20, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "decorated_pot"})

	if err := handleUseItemOn(useItemOnPacket(20, 64, 0, 1, 505), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	items := w.ContainerItems(20, 64, 0)
	if len(items) != 1 || items[0].ItemID != "minecraft:diamond" || items[0].Count != 1 {
		t.Fatalf("pot items = %+v", items)
	}
	if p.HeldItem().Count != 1 {
		t.Fatalf("held count = %d, want 1", p.HeldItem().Count)
	}
}

func TestJavaButtonUsesClickedFaceAndStaysAttached(t *testing.T) {
	p := player.New([16]byte{}, "switcher", player.ClientEditionJava)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:stone_button", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(22, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	if err := handleUseItemOn(useItemOnPacket(22, 64, 0, 5, 506), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	button := w.GetBlock(23, 64, 0)
	if button.ResourceLocation() != "minecraft:stone_button" || button.Properties["face"] != "wall" ||
		button.Properties["facing"] != "east" || button.Properties["powered"] != "false" {
		t.Fatalf("button state = %s", button.Key())
	}
}

func TestJavaBoneMealGrowsCropAndConsumesItem(t *testing.T) {
	p := player.New([16]byte{}, "farmer", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:bone_meal", Count: 2}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(24, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "wheat", Properties: map[string]string{"age": "0"}})

	if err := handleUseItemOn(useItemOnPacket(24, 64, 0, 1, 507), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(24, 64, 0).Properties["age"]; got != "7" {
		t.Fatalf("wheat age = %q, want 7", got)
	}
	if got := p.HeldItem().Count; got != 1 {
		t.Fatalf("bone meal count = %d, want 1", got)
	}
}
