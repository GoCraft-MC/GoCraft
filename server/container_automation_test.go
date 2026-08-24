package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
	"GoCraft/java/session"
)

func newAutomationTestServer(t *testing.T) (*Server, *coreworld.World) {
	t.Helper()
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = world.Close() })
	return &Server{world: world, game: game.New(), sessions: session.NewManager(), worldAge: 8}, world
}

func TestHopperPushesOneItemIntoContainerEveryCycle(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 65, 0, bedrockBlock("hopper", map[string]string{"facing": "down", "enabled": "true"}))
	world.SetContainerItems(0, 65, 0, "minecraft:hopper", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:diamond", Count: 2}})
	world.SetBlock(0, 64, 0, bedrockBlock("chest", nil))
	world.SetContainerItems(0, 64, 0, "minecraft:chest", nil)

	s.tickContainerAutomation()
	hopper, chest := world.ContainerItems(0, 65, 0), world.ContainerItems(0, 64, 0)
	if len(hopper) != 1 || hopper[0].Count != 1 || len(chest) != 1 || chest[0].ItemID != "minecraft:diamond" || chest[0].Count != 1 {
		t.Fatalf("hopper/chest after push = %+v / %+v", hopper, chest)
	}
}

func TestHopperPullsOneItemFromContainerAbove(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("hopper", map[string]string{"facing": "east", "enabled": "true"}))
	world.SetContainerItems(0, 64, 0, "minecraft:hopper", nil)
	world.SetBlock(0, 65, 0, bedrockBlock("barrel", nil))
	world.SetContainerItems(0, 65, 0, "minecraft:barrel", []coreworld.ContainerItem{{Slot: 4, ItemID: "minecraft:apple", Count: 3}})

	s.tickContainerAutomation()
	hopper, barrel := world.ContainerItems(0, 64, 0), world.ContainerItems(0, 65, 0)
	if len(hopper) != 1 || hopper[0].ItemID != "minecraft:apple" || hopper[0].Count != 1 || len(barrel) != 1 || barrel[0].Count != 2 {
		t.Fatalf("hopper/barrel after pull = %+v / %+v", hopper, barrel)
	}
}

func TestHopperCollectsDroppedItemAboveIt(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("hopper", map[string]string{"facing": "down", "enabled": "true"}))
	world.SetContainerItems(0, 64, 0, "minecraft:hopper", nil)
	dropped := corentity.New(s.game.NextEntityID(), [16]byte{22}, corentity.TypeItem, 0.5, 65.2, 0.5)
	dropped.ItemID = "minecraft:emerald"
	dropped.ItemCount = 2
	world.Entities.Add(dropped)

	s.tickContainerAutomation()
	items := world.ContainerItems(0, 64, 0)
	if len(items) != 1 || items[0].ItemID != "minecraft:emerald" || items[0].Count != 1 {
		t.Fatalf("hopper did not collect dropped item: %+v", items)
	}
	if dropped.ItemCount != 1 {
		t.Fatalf("dropped stack count = %d, want 1", dropped.ItemCount)
	}
}

func TestPoweredHopperIsLocked(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 65, 0, bedrockBlock("hopper", map[string]string{"facing": "down", "enabled": "true"}))
	world.SetContainerItems(0, 65, 0, "minecraft:hopper", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:diamond", Count: 1}})
	world.SetBlock(0, 64, 0, bedrockBlock("chest", nil))
	world.SetContainerItems(0, 64, 0, "minecraft:chest", nil)
	world.SetBlock(1, 65, 0, bedrockBlock("redstone_block", nil))
	world.Redstone.FlushUpdates()

	s.tickContainerAutomation()
	if chest := world.ContainerItems(0, 64, 0); len(chest) != 0 {
		t.Fatalf("powered hopper transferred into chest: %+v", chest)
	}
	if enabled := world.GetBlock(0, 65, 0).Properties["enabled"]; enabled != "false" {
		t.Fatalf("powered hopper enabled state = %q", enabled)
	}
}

func TestPoweredDropperInsertsOneItemIntoFacingContainer(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("dropper", map[string]string{"facing": "east", "triggered": "false"}))
	world.SetContainerItems(0, 64, 0, "minecraft:dropper", []coreworld.ContainerItem{{Slot: 2, ItemID: "minecraft:cobblestone", Count: 2}})
	world.SetBlock(1, 64, 0, bedrockBlock("chest", nil))
	world.SetContainerItems(1, 64, 0, "minecraft:chest", nil)
	world.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	dropper, chest := world.ContainerItems(0, 64, 0), world.ContainerItems(1, 64, 0)
	if len(dropper) != 1 || dropper[0].Count != 1 || len(chest) != 1 || chest[0].ItemID != "minecraft:cobblestone" {
		t.Fatalf("dropper/chest after pulse = %+v / %+v", dropper, chest)
	}
}

func TestPoweredDispenserFiresArrowProjectile(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("dispenser", map[string]string{"facing": "east", "triggered": "false"}))
	world.SetContainerItems(0, 64, 0, "minecraft:dispenser", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:arrow", Count: 1}})
	world.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	entities := world.Entities.Snapshot()
	if len(entities) != 1 || entities[0].Type != corentity.TypeArrow || entities[0].VX <= 0 {
		t.Fatalf("dispenser entities = %+v", entities)
	}
	if items := world.ContainerItems(0, 64, 0); len(items) != 0 {
		t.Fatalf("dispenser arrow was not consumed: %+v", items)
	}
}

func TestPoweredCrafterCraftsIntoFacingContainer(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("crafter", map[string]string{"orientation": "east_up", "crafting": "false", "triggered": "false"}))
	world.SetContainerItems(0, 64, 0, "minecraft:crafter", []coreworld.ContainerItem{{Slot: 4, ItemID: "minecraft:oak_log", Count: 1}})
	world.SetBlock(1, 64, 0, bedrockBlock("chest", nil))
	world.SetContainerItems(1, 64, 0, "minecraft:chest", nil)
	world.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	if got := world.ContainerItems(0, 64, 0); len(got) != 0 {
		t.Fatalf("crafter ingredient was not consumed: %+v", got)
	}
	if got := world.ContainerItems(1, 64, 0); len(got) != 1 || got[0].ItemID != "minecraft:oak_planks" || got[0].Count != 4 {
		t.Fatalf("crafter output = %+v, want four oak planks", got)
	}
}

func TestNetherRedstoneActivatesDropperInNetherWorld(t *testing.T) {
	s, _ := newAutomationTestServer(t)
	nether := coreworld.New(&coreworld.NetherGenerator{}, nil, false)
	t.Cleanup(func() { _ = nether.Close() })
	s.netherWorld = nether
	nether.SetBlock(0, 64, 0, bedrockBlock("dropper", map[string]string{"facing": "east", "triggered": "false"}))
	nether.SetContainerItems(0, 64, 0, "minecraft:dropper", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:quartz", Count: 1}})
	nether.SetBlock(1, 64, 0, bedrockBlock("chest", nil))
	nether.SetContainerItems(1, 64, 0, "minecraft:chest", nil)
	nether.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	if got := nether.ContainerItems(1, 64, 0); len(got) != 1 || got[0].ItemID != "minecraft:quartz" {
		t.Fatalf("Nether dropper did not activate in its own world: %+v", got)
	}
	if got := s.world.ContainerItems(1, 64, 0); len(got) != 0 {
		t.Fatalf("Nether automation leaked into Overworld: %+v", got)
	}
}

func TestDispenserBucketCollectsWaterAndKeepsFilledBucket(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("dispenser", map[string]string{"facing": "east", "triggered": "false"}))
	world.SetContainerItems(0, 64, 0, "minecraft:dispenser", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:bucket", Count: 1}})
	world.SetBlock(1, 64, 0, coreworld.MakeFluid("minecraft:water", 0))
	world.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	items := world.ContainerItems(0, 64, 0)
	if len(items) != 1 || items[0].ItemID != "minecraft:water_bucket" || items[0].Count != 1 {
		t.Fatalf("dispenser bucket result = %+v", items)
	}
	if got := world.GetBlock(1, 64, 0); !got.IsAir() {
		t.Fatalf("collected water remains at target: %+v", got)
	}
}

func TestDispenserFiresSnowballProjectile(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("dispenser", map[string]string{"facing": "east", "triggered": "false"}))
	world.SetContainerItems(0, 64, 0, "minecraft:dispenser", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:snowball", Count: 1}})
	world.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	entities := world.Entities.Snapshot()
	if len(entities) != 1 || entities[0].Type != corentity.TypeSnowball || entities[0].VX <= 0 {
		t.Fatalf("snowball dispenser entities = %+v", entities)
	}
}

func TestDispenserUsesSpawnEgg(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("dispenser", map[string]string{"facing": "east", "triggered": "false"}))
	world.SetContainerItems(0, 64, 0, "minecraft:dispenser", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:cow_spawn_egg", Count: 1}})
	world.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	for _, entity := range world.Entities.Snapshot() {
		if entity.Type == corentity.TypeCow {
			return
		}
	}
	t.Fatal("spawn egg dispenser spawned no cow")
}

func TestDispenserAppliesBoneMealToCrop(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("dispenser", map[string]string{"facing": "east", "triggered": "false"}))
	world.SetContainerItems(0, 64, 0, "minecraft:dispenser", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:bone_meal", Count: 1}})
	world.SetBlock(1, 64, 0, bedrockBlock("wheat", map[string]string{"age": "0"}))
	world.SetBlock(0, 64, -1, bedrockBlock("redstone_block", nil))

	s.tickBlockPhysics()
	if got := world.GetBlock(1, 64, 0).Properties["age"]; got == "" || got == "0" {
		t.Fatalf("bone meal dispenser left wheat age at %q", got)
	}
}

func TestCampfireCooksAndDropsFoodAfterRecipeDuration(t *testing.T) {
	s, world := newAutomationTestServer(t)
	world.SetBlock(0, 64, 0, bedrockBlock("campfire", map[string]string{"lit": "true"}))
	world.SetContainerItems(0, 64, 0, "minecraft:campfire", []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:beef", Count: 1}})
	s.worldAge = 8
	s.tickCampfires(world, dimensionOverworld)
	recipe, ok := handler.FindCookingRecipe("minecraft:campfire", "minecraft:beef")
	if !ok {
		t.Fatal("missing embedded campfire beef recipe")
	}
	s.worldAge += int64(recipe.CookingTime)
	s.tickCampfires(world, dimensionOverworld)
	if got := world.ContainerItems(0, 64, 0); len(got) != 0 {
		t.Fatalf("campfire still contains %+v", got)
	}
	for _, entity := range world.Entities.Snapshot() {
		if entity.Type == corentity.TypeItem && entity.ItemID == "minecraft:cooked_beef" && entity.ItemCount == 1 {
			return
		}
	}
	t.Fatal("campfire produced no cooked beef drop")
}
