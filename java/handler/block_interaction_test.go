package handler

import (
	"strconv"
	"strings"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
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
	if !javaWorldHasDroppedItem(w, "minecraft:dirt", 1) {
		t.Fatal("survival break did not spawn one dirt item")
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
	if !javaWorldHasDroppedItem(w, "minecraft:peony", 1) {
		t.Fatal("flower break did not spawn one peony item")
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
	if got := p.Inventory[player.HotbarStart]; !got.IsEmpty() {
		t.Fatalf("grass drop went directly to inventory: %+v", got)
	}
	for _, entity := range w.Entities.Snapshot() {
		if entity.Type == corentity.TypeItem && (entity.ItemID != "minecraft:wheat_seeds" || entity.ItemCount != 1) {
			t.Fatalf("grass drop = %+v, want one wheat seed", entity)
		}
	}
}

func javaWorldHasDroppedItem(w *coreworld.World, itemID string, count int) bool {
	for _, entity := range w.Entities.Snapshot() {
		if entity.Type == corentity.TypeItem && entity.ItemID == itemID && entity.ItemCount == count {
			return true
		}
	}
	return false
}

func TestJavaPlayerDropActionsPreserveStacks(t *testing.T) {
	p := player.New([16]byte{}, "dropper", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:diamond", Count: 3, Components: `{"custom_name":"Gem"}`}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	nextID := int32(0)
	for _, status := range []int32{actionStatusDropItem, actionStatusDropStack} {
		pkt := protocol.NewBuilder(packetIDPlayerAction).
			VarInt(status).Long(packBlockPos(0, 64, 0)).Byte(1).VarInt(status).Build()
		if err := handlePlayerActionWithContext(pkt, p, w, session.NewManager(), nil, func() int32 {
			nextID++
			return nextID
		}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if stack := p.Inventory[player.HotbarStart]; !stack.IsEmpty() {
		t.Fatalf("held stack remains: %+v", stack)
	}
	counts := map[int]bool{}
	for _, entity := range w.Entities.Snapshot() {
		stack := entity.DroppedItem()
		if stack.ItemID == "minecraft:diamond" && stack.Components == `{"custom_name":"Gem"}` {
			counts[stack.Count] = true
		}
	}
	if !counts[1] || !counts[2] {
		t.Fatalf("dropped stack counts = %+v", counts)
	}
}

func TestJavaPlayerActionSwapsOffhand(t *testing.T) {
	p := player.New([16]byte{}, "swapper", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:torch", Count: 4}
	p.Inventory[player.OffhandSlot] = player.ItemStack{ItemID: "minecraft:shield", Count: 1}
	pkt := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusSwapOffhand).Long(packBlockPos(0, 64, 0)).Byte(1).VarInt(1).Build()
	if err := handlePlayerAction(pkt, p, nil, session.NewManager()); err != nil {
		t.Fatal(err)
	}
	if p.Inventory[player.HotbarStart].ItemID != "minecraft:shield" || p.Inventory[player.OffhandSlot].ItemID != "minecraft:torch" {
		t.Fatalf("swapped inventory = main %+v offhand %+v", p.Inventory[player.HotbarStart], p.Inventory[player.OffhandSlot])
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

func TestJavaRedstoneWireCannotStackOrFloat(t *testing.T) {
	p := player.New([16]byte{}, "engineer", player.ClientEditionJava)
	p.GameMode = player.GameModeCreative
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:redstone", Count: 64}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(30, 70, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	if err := handleUseItemOn(useItemOnPacket(30, 70, 0, 1, 510), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := handleUseItemOn(useItemOnPacket(30, 71, 0, 1, 511), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := w.GetBlock(30, 72, 0); !got.IsAir() {
		t.Fatalf("wire placed on wire = %s, want air", got.Key())
	}
}

func TestJavaRedstoneWireRefreshesBothSidesAndSteps(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(1, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "redstone_wire", Properties: redstoneWireConnections(0, 64, 0, w)})
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "redstone_wire", Properties: redstoneWireConnections(1, 64, 0, w)})
	refreshJavaConnectedBlocks(1, 64, 0, w, nil)
	if left, right := w.GetBlock(0, 64, 0), w.GetBlock(1, 64, 0); left.Properties["east"] != "side" || right.Properties["west"] != "side" {
		t.Fatalf("horizontal wire states left=%v right=%v", left.Properties, right.Properties)
	}

	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(1, 65, 0, coreworld.Block{Namespace: "minecraft", Name: "redstone_wire", Properties: map[string]string{"power": "0"}})
	refreshJavaConnectedBlocks(1, 64, 0, w, nil)
	if lower, upper := w.GetBlock(0, 64, 0), w.GetBlock(1, 65, 0); lower.Properties["east"] != "up" || upper.Properties["west"] != "side" {
		t.Fatalf("stepped wire states lower=%v upper=%v", lower.Properties, upper.Properties)
	}
}

func TestJavaRepeaterAndWoodPlacementStates(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	p.GameMode = player.GameModeCreative
	p.Rotation.Yaw = 0 // looking south; repeater input faces back north
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(40, 70, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:repeater", Count: 64}
	if err := handleUseItemOn(useItemOnPacket(40, 70, 0, 1, 512), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	repeater := w.GetBlock(40, 71, 0)
	if repeater.Properties["facing"] != "north" || javaworld.StateID(repeater) == 0 {
		t.Fatalf("repeater state = %s (id=%d)", repeater.Key(), javaworld.StateID(repeater))
	}

	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 64}
	if err := handleUseItemOn(useItemOnPacket(40, 70, 0, 5, 513), p, w, mgr, nil, nil); err != nil {
		t.Fatal(err)
	}
	log := w.GetBlock(41, 70, 0)
	if log.Properties["axis"] != "x" || javaworld.StateID(log) == 0 {
		t.Fatalf("side-placed log state = %s (id=%d)", log.Key(), javaworld.StateID(log))
	}
}

func TestJavaPlacementUsesExactAdjacentBlockForEveryFace(t *testing.T) {
	for face, offset := range faceOffset {
		t.Run(strconv.Itoa(face), func(t *testing.T) {
			p := player.New([16]byte{}, "builder", player.ClientEditionJava)
			p.GameMode = player.GameModeCreative
			p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 64}
			w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer w.Close()
			w.SetBlock(50, 70, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
			if err := handleUseItemOn(useItemOnPacket(50, 70, 0, int32(face), int32(520+face)), p, w, session.NewManager(), nil, nil); err != nil {
				t.Fatal(err)
			}
			x, y, z := 50+int(offset[0]), 70+int(offset[1]), int(offset[2])
			if got := w.GetBlock(x, y, z); got.ResourceLocation() != "minecraft:oak_log" {
				t.Fatalf("face %d placed %q at (%d,%d,%d)", face, got.ResourceLocation(), x, y, z)
			}
		})
	}
}

func TestJavaBedWakeUsesBedPositionAndFacing(t *testing.T) {
	p := player.New([16]byte{}, "sleeper", player.ClientEditionJava)
	p.Position = spatial.Vec3{X: 20, Y: 80, Z: 20}
	p.Rotation.Yaw = 180
	p.SpawnPoint = spatial.BlockPos{X: 0, Y: 64, Z: 0}
	p.HasSpawnPoint = true
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{"part": "foot", "facing": "south"}})
	w.SetBlock(0, 64, 1, coreworld.Block{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{"part": "head", "facing": "south"}})
	w.SetBlock(-1, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	if !prepareJavaBedWake(p, w, nil) {
		t.Fatal("bed wake state was not resolved")
	}
	if p.Position != (spatial.Vec3{X: -0.5, Y: 64, Z: 0.5}) || p.Rotation.Yaw != 0 || p.Rotation.Pitch != 0 {
		t.Fatalf("wake state position=%+v rotation=%+v", p.Position, p.Rotation)
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

func TestJavaImmediateFluidInteractionDirections(t *testing.T) {
	tests := []struct {
		name              string
		placed, neighbor  string
		dx, dy            int
		wantPlaced, wantN string
	}{
		{"water above lava", "lava", "water", 0, 1, "minecraft:obsidian", "minecraft:water"},
		{"water below lava", "lava", "water", 0, -1, "minecraft:lava", "minecraft:water"},
		{"lava above water", "water", "lava", 0, 1, "minecraft:water", "minecraft:lava"},
		{"lava below water", "water", "lava", 0, -1, "minecraft:water", "minecraft:obsidian"},
		{"flowing lava beside water", "lava", "water", 1, 0, "minecraft:cobblestone", "minecraft:water"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer w.Close()
			level := 0
			if test.name == "flowing lava beside water" {
				level = 2
			}
			w.SetBlock(0, 65, 0, coreworld.MakeFluid("minecraft:"+test.placed, level))
			w.SetBlock(test.dx, 65+test.dy, 0, coreworld.MakeFluid("minecraft:"+test.neighbor, 0))
			checkFluidInteraction(0, 65, 0, w, nil)
			if got := w.GetBlock(0, 65, 0).ResourceLocation(); got != test.wantPlaced {
				t.Errorf("placed block = %s, want %s", got, test.wantPlaced)
			}
			if got := w.GetBlock(test.dx, 65+test.dy, 0).ResourceLocation(); got != test.wantN {
				t.Errorf("neighbor block = %s, want %s", got, test.wantN)
			}
		})
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

func TestJavaActivatesRedstoneControls(t *testing.T) {
	p := player.New([16]byte{}, "switcher", player.ClientEditionJava)
	p.GameMode = player.GameModeCreative
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(25, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone_button", Properties: map[string]string{"powered": "false"}})
	w.SetBlock(26, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "repeater", Properties: map[string]string{
		"delay": "1", "facing": "north", "locked": "false", "powered": "false",
	}})
	w.SetBlock(27, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "comparator", Properties: map[string]string{
		"facing": "north", "mode": "compare", "powered": "false",
	}})

	for index, x := range []int{25, 26, 27} {
		if err := handleUseItemOn(useItemOnPacket(x, 64, 0, 1, int32(530+index)), p, w, mgr, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := w.GetBlock(25, 64, 0).Properties["powered"]; got != "true" {
		t.Fatalf("button powered = %q", got)
	}
	if got := w.GetBlock(26, 64, 0).Properties["delay"]; got != "2" {
		t.Fatalf("repeater delay = %q", got)
	}
	if got := w.GetBlock(27, 64, 0).Properties["mode"]; got != "subtract" {
		t.Fatalf("comparator mode = %q", got)
	}
}

func TestJavaPlacesLitRedstoneWallTorch(t *testing.T) {
	p := player.New([16]byte{}, "engineer", player.ClientEditionJava)
	p.GameMode = player.GameModeCreative
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:redstone_torch", Count: 64}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(30, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	if err := handleUseItemOn(useItemOnPacket(30, 64, 0, 5, 540), p, w, session.NewManager(), nil, nil); err != nil {
		t.Fatal(err)
	}
	torch := w.GetBlock(31, 64, 0)
	if torch.ResourceLocation() != "minecraft:redstone_wall_torch" ||
		torch.Properties["facing"] != "east" || torch.Properties["lit"] != "true" {
		t.Fatalf("wall torch state = %s", torch.Key())
	}
}

func TestJavaPlacesMinecartOnRail(t *testing.T) {
	p := player.New([16]byte{}, "driver", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:minecart", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(32, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "rail", Properties: map[string]string{"shape": "east_west"}})
	if err := handleUseItemOn(useItemOnPacket(32, 64, 0, 1, 541), p, w, session.NewManager(), nil, func() int32 { return 77 }); err != nil {
		t.Fatal(err)
	}
	entity, ok := w.Entities.Get(77)
	if !ok || entity.Type != corentity.TypeMinecart || entity.Position.X != 32.5 {
		t.Fatalf("placed minecart = %+v, exists=%v", entity, ok)
	}
	if !p.HeldItem().IsEmpty() {
		t.Fatalf("held item after placement = %+v", p.HeldItem())
	}
}

func TestJavaPlacesWeightedPressurePlateState(t *testing.T) {
	p := player.New([16]byte{}, "engineer", player.ClientEditionJava)
	p.GameMode = player.GameModeCreative
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:heavy_weighted_pressure_plate", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(34, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	if err := handleUseItemOn(useItemOnPacket(34, 63, 0, 1, 542), p, w, session.NewManager(), nil, nil); err != nil {
		t.Fatal(err)
	}
	plate := w.GetBlock(34, 64, 0)
	if plate.ResourceLocation() != "minecraft:heavy_weighted_pressure_plate" || plate.Properties["power"] != "0" {
		t.Fatalf("pressure plate state = %s", plate.Key())
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
	// Bone meal advances wheat by 2-5 stages (vanilla behaviour). Starting from
	// age 0 the result is somewhere between 2 and 5; it must be greater than 0
	// and no greater than the maxAge of 7.
	gotAge := w.GetBlock(24, 64, 0).Properties["age"]
	if gotAge == "0" || gotAge == "1" {
		t.Fatalf("wheat age = %q, want >= 2 (bone meal must advance crop)", gotAge)
	}
	if got := p.HeldItem().Count; got != 1 {
		t.Fatalf("bone meal count = %d, want 1", got)
	}
}

func TestJavaPlantsEverySupportedCrop(t *testing.T) {
	tests := []struct {
		item, support, crop string
	}{
		{item: "minecraft:wheat_seeds", support: "farmland", crop: "minecraft:wheat"},
		{item: "minecraft:carrot", support: "farmland", crop: "minecraft:carrots"},
		{item: "minecraft:potato", support: "farmland", crop: "minecraft:potatoes"},
		{item: "minecraft:beetroot_seeds", support: "farmland", crop: "minecraft:beetroots"},
		{item: "minecraft:melon_seeds", support: "farmland", crop: "minecraft:melon_stem"},
		{item: "minecraft:pumpkin_seeds", support: "farmland", crop: "minecraft:pumpkin_stem"},
		{item: "minecraft:torchflower_seeds", support: "farmland", crop: "minecraft:torchflower_crop"},
		{item: "minecraft:nether_wart", support: "soul_sand", crop: "minecraft:nether_wart"},
	}
	for index, test := range tests {
		t.Run(test.item, func(t *testing.T) {
			p := player.New([16]byte{byte(index + 1)}, "farmer", player.ClientEditionJava)
			p.GameMode = player.GameModeSurvival
			p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: test.item, Count: 1}
			w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer w.Close()
			mgr := session.NewManager()
			w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: test.support, Properties: map[string]string{"moisture": "0"}})

			if err := handleUseItemOn(useItemOnPacket(0, 64, 0, 1, int32(600+index)), p, w, mgr, nil, nil); err != nil {
				t.Fatal(err)
			}
			placed := w.GetBlock(0, 65, 0)
			if placed.ResourceLocation() != test.crop || coreworld.CropAge(placed) != 0 {
				t.Fatalf("placed crop = %+v, want %s age 0", placed, test.crop)
			}
			if !p.HeldItem().IsEmpty() {
				t.Fatalf("seed was not consumed: %+v", p.HeldItem())
			}
		})
	}
}

func TestJavaNetherWartRejectsBoneMeal(t *testing.T) {
	p := player.New([16]byte{90}, "wart-farmer", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:bone_meal", Count: 2}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "soul_sand"})
	w.SetBlock(0, 65, 0, coreworld.Block{Namespace: "minecraft", Name: "nether_wart", Properties: map[string]string{"age": "0"}})
	if err := handleUseItemOn(useItemOnPacket(0, 65, 0, 1, 700), p, w, session.NewManager(), nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := coreworld.CropAge(w.GetBlock(0, 65, 0)); got != 0 {
		t.Fatalf("nether wart age = %d, want 0", got)
	}
	if got := p.HeldItem().Count; got != 2 {
		t.Fatalf("bone meal count = %d, want 2", got)
	}
}

func TestJavaHarvestsSweetBerryBush(t *testing.T) {
	p := player.New([16]byte{91}, "berry-picker", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "dirt"})
	w.SetBlock(0, 65, 0, coreworld.Block{Namespace: "minecraft", Name: "sweet_berry_bush", Properties: map[string]string{"age": "3"}})
	if err := handleUseItemOn(useItemOnPacket(0, 65, 0, 1, 701), p, w, session.NewManager(), nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := coreworld.CropAge(w.GetBlock(0, 65, 0)); got != 1 {
		t.Fatalf("harvested bush age = %d, want 1", got)
	}
	berries := 0
	for _, stack := range p.Inventory {
		if stack.ItemID == "minecraft:sweet_berries" {
			berries += stack.Count
		}
	}
	if berries < 2 || berries > 3 {
		t.Fatalf("harvest berries = %d, want 2..3", berries)
	}
}

func TestJavaPlacesFoodOnCampfire(t *testing.T) {
	p := player.New([16]byte{92}, "camper", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:beef", Count: 2}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "campfire", Properties: map[string]string{"lit": "true"}})
	if err := handleUseItemOn(useItemOnPacket(0, 64, 0, 1, 702), p, w, session.NewManager(), nil, nil); err != nil {
		t.Fatal(err)
	}
	items := w.ContainerItems(0, 64, 0)
	if len(items) != 1 || items[0].ItemID != "minecraft:beef" || items[0].Count != 1 {
		t.Fatalf("campfire items = %+v", items)
	}
	if got := p.HeldItem().Count; got != 1 {
		t.Fatalf("held beef = %d, want 1", got)
	}
}
