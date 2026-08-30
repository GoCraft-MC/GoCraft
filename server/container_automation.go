package server

import (
	"math"
	"sort"
	"strings"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

// tickContainerAutomation runs Vanilla's eight-tick hopper cadence in every
// dimension. Only loaded block entities are scanned, so automation never
// generates distant chunks by itself.
func (s *Server) tickContainerAutomation() {
	if s == nil || s.worldAge%8 != 0 {
		return
	}
	for dimension, world := range map[int32]*coreworld.World{
		dimensionOverworld: s.world,
		dimensionNether:    s.netherWorld,
		dimensionEnd:       s.endWorld,
	} {
		if world != nil {
			s.tickHoppers(world, dimension)
			s.tickCampfires(world, dimension)
		}
	}
}

type campfireCookKey struct {
	dimension int32
	x, y, z   int
	slot      int
}

func (s *Server) tickCampfires(world *coreworld.World, dimension int32) {
	if s.campfireCooking == nil {
		s.campfireCooking = make(map[campfireCookKey]int64)
	}
	active := make(map[campfireCookKey]struct{})
	for _, blockEntity := range world.LoadedBlockEntities() {
		block := world.GetBlock(blockEntity.X, blockEntity.Y, blockEntity.Z)
		name := block.ResourceLocation()
		if (name != "minecraft:campfire" && name != "minecraft:soul_campfire") || block.Properties["lit"] == "false" {
			continue
		}
		items := world.ContainerItems(blockEntity.X, blockEntity.Y, blockEntity.Z)
		changed := false
		for index := range items {
			item := items[index]
			recipe, ok := handler.FindCookingRecipe("minecraft:campfire", item.ItemID)
			if !ok {
				continue
			}
			key := campfireCookKey{dimension: dimension, x: blockEntity.X, y: blockEntity.Y, z: blockEntity.Z, slot: item.Slot}
			active[key] = struct{}{}
			started, exists := s.campfireCooking[key]
			if !exists {
				s.campfireCooking[key] = s.worldAge
				continue
			}
			if s.worldAge-started < int64(recipe.CookingTime) {
				continue
			}
			delete(s.campfireCooking, key)
			items[index].Count--
			changed = true
			position := spatial.Vec3{X: float64(blockEntity.X) + 0.5, Y: float64(blockEntity.Y) + 1.1, Z: float64(blockEntity.Z) + 0.5}
			if dropped := s.newDroppedItemInWorld(world, recipe.Result, position, item.Slot); dropped != nil {
				handler.BroadcastSpawnMob(dropped, s.javaSessionsForDimension(dimension))
			}
		}
		if changed {
			world.SetContainerItems(blockEntity.X, blockEntity.Y, blockEntity.Z, name, items)
		}
	}
	for key := range s.campfireCooking {
		if key.dimension == dimension {
			if _, ok := active[key]; !ok {
				delete(s.campfireCooking, key)
			}
		}
	}
}

func (s *Server) tickHoppers(world *coreworld.World, dimension int32) {
	blockEntities := world.LoadedBlockEntities()
	sort.Slice(blockEntities, func(i, j int) bool {
		if blockEntities[i].Y != blockEntities[j].Y {
			return blockEntities[i].Y < blockEntities[j].Y
		}
		if blockEntities[i].X != blockEntities[j].X {
			return blockEntities[i].X < blockEntities[j].X
		}
		return blockEntities[i].Z < blockEntities[j].Z
	})
	receivedThisTick := make(map[[3]int]struct{})
	for _, entity := range blockEntities {
		position := [3]int{entity.X, entity.Y, entity.Z}
		block := world.GetBlock(entity.X, entity.Y, entity.Z)
		if block.ResourceLocation() != "minecraft:hopper" {
			continue
		}
		powered := world.Redstone.PowerAt(entity.X, entity.Y, entity.Z) > 0
		s.setHopperEnabled(world, dimension, entity.X, entity.Y, entity.Z, block, !powered)
		if powered {
			continue
		}
		if _, justReceived := receivedThisTick[position]; justReceived {
			continue
		}

		dx, dy, dz := containerFacingOffset(block.Properties["facing"])
		destination := [3]int{entity.X + dx, entity.Y + dy, entity.Z + dz}
		if s.transferOneContainerItem(world, position, destination) {
			if world.GetBlock(destination[0], destination[1], destination[2]).ResourceLocation() == "minecraft:hopper" {
				receivedThisTick[destination] = struct{}{}
			}
			s.syncAutomatedContainers(world, dimension, position, destination)
			continue
		}

		source := [3]int{entity.X, entity.Y + 1, entity.Z}
		if s.transferOneContainerItem(world, source, position) {
			receivedThisTick[position] = struct{}{}
			s.syncAutomatedContainers(world, dimension, source, position)
			continue
		}
		if s.collectDroppedItemIntoHopper(world, dimension, position) {
			receivedThisTick[position] = struct{}{}
			s.syncAutomatedContainers(world, dimension, position)
		}
	}
}

func (s *Server) collectDroppedItemIntoHopper(world *coreworld.World, dimension int32, hopperPosition [3]int) bool {
	kind, stacks, ok := loadAutomationContainer(world, hopperPosition)
	if !ok || kind != "minecraft:hopper" {
		return false
	}
	centreX, topY, centreZ := float64(hopperPosition[0])+0.5, float64(hopperPosition[1])+1, float64(hopperPosition[2])+0.5
	for _, entity := range world.Entities.Snapshot() {
		if entity.Dead || entity.Type != corentity.TypeItem || entity.ItemID == "" || entity.ItemCount <= 0 ||
			math.Abs(entity.Position.X-centreX) > 0.75 || entity.Position.Y < topY-0.5 || entity.Position.Y > topY+1.5 ||
			math.Abs(entity.Position.Z-centreZ) > 0.75 {
			continue
		}
		item := player.ItemStack{ItemID: entity.ItemID, Count: 1, Damage: entity.ItemDamage}
		if !insertOneAutomationItem(kind, stacks, item, hopperPosition, [3]int{hopperPosition[0], hopperPosition[1] + 1, hopperPosition[2]}) {
			continue
		}
		saveAutomationContainer(world, hopperPosition, kind, stacks)
		entity.ItemCount--
		if entity.ItemCount <= 0 {
			world.Entities.Remove(entity.EntityID)
			handler.BroadcastRemoveEntity(entity.EntityID, s.javaSessionsForDimension(dimension))
		}
		return true
	}
	return false
}

func (s *Server) setHopperEnabled(world *coreworld.World, dimension int32, x, y, z int, block coreworld.Block, enabled bool) {
	current := block.Properties["enabled"] != "false"
	if current == enabled {
		return
	}
	replacement := bedrockCopyBlock(block)
	replacement.Properties["enabled"] = strconvFormatBool(enabled)
	world.SetBlock(x, y, z, replacement)
	handler.BroadcastBlockChange(coreworld.BlockChange{X: x, Y: y, Z: z, Block: replacement}, s.javaSessionsForDimension(dimension))
}

func strconvFormatBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func automationContainerSize(kind string) int {
	switch kind {
	case "minecraft:hopper":
		return 5
	case "minecraft:dispenser", "minecraft:dropper", "minecraft:crafter":
		return 9
	case "minecraft:furnace", "minecraft:blast_furnace", "minecraft:smoker":
		return 3
	case "minecraft:brewing_stand":
		return 5
	case "minecraft:decorated_pot":
		return 1
	case "minecraft:chest", "minecraft:trapped_chest", "minecraft:barrel":
		return 27
	default:
		if strings.HasSuffix(kind, "_shulker_box") || kind == "minecraft:shulker_box" {
			return 27
		}
		return 0
	}
}

func loadAutomationContainer(world *coreworld.World, position [3]int) (string, []player.ItemStack, bool) {
	kind := world.GetBlock(position[0], position[1], position[2]).ResourceLocation()
	size := automationContainerSize(kind)
	if size == 0 {
		return "", nil, false
	}
	stacks := make([]player.ItemStack, size)
	for _, item := range world.ContainerItems(position[0], position[1], position[2]) {
		if item.Slot >= 0 && item.Slot < size && item.ItemID != "" && item.Count > 0 {
			stacks[item.Slot] = player.ItemStack{ItemID: item.ItemID, Count: item.Count, Damage: item.Damage, Enchantments: item.Enchantments, PotDecorations: item.PotDecorations}
		}
	}
	return kind, stacks, true
}

func saveAutomationContainer(world *coreworld.World, position [3]int, kind string, stacks []player.ItemStack) {
	items := make([]coreworld.ContainerItem, 0, len(stacks))
	for slot, stack := range stacks {
		if !stack.IsEmpty() {
			items = append(items, coreworld.ContainerItem{Slot: slot, ItemID: stack.ItemID, Count: stack.Count, Damage: stack.Damage, Enchantments: stack.Enchantments, PotDecorations: stack.PotDecorations})
		}
	}
	world.SetContainerItems(position[0], position[1], position[2], kind, items)
}

func (s *Server) transferOneContainerItem(world *coreworld.World, sourcePosition, destinationPosition [3]int) bool {
	sourceKind, source, sourceOK := loadAutomationContainer(world, sourcePosition)
	destinationKind, destination, destinationOK := loadAutomationContainer(world, destinationPosition)
	if !sourceOK || !destinationOK {
		return false
	}
	sourceSlots := automationExtractionSlots(sourceKind, len(source))
	for _, sourceSlot := range sourceSlots {
		if sourceSlot < 0 || sourceSlot >= len(source) || source[sourceSlot].IsEmpty() {
			continue
		}
		item := source[sourceSlot]
		if !insertOneAutomationItem(destinationKind, destination, item, destinationPosition, sourcePosition) {
			continue
		}
		source[sourceSlot].Count--
		if source[sourceSlot].Count <= 0 {
			source[sourceSlot] = player.ItemStack{}
		}
		saveAutomationContainer(world, sourcePosition, sourceKind, source)
		saveAutomationContainer(world, destinationPosition, destinationKind, destination)
		return true
	}
	return false
}

func automationExtractionSlots(kind string, size int) []int {
	if kind == "minecraft:furnace" || kind == "minecraft:blast_furnace" || kind == "minecraft:smoker" {
		return []int{2}
	}
	slots := make([]int, size)
	for slot := range slots {
		slots[slot] = slot
	}
	return slots
}

func insertOneAutomationItem(kind string, destination []player.ItemStack, item player.ItemStack, destinationPosition, sourcePosition [3]int) bool {
	slots := make([]int, len(destination))
	for index := range slots {
		slots[index] = index
	}
	if kind == "minecraft:furnace" || kind == "minecraft:blast_furnace" || kind == "minecraft:smoker" {
		if destinationPosition[1] < sourcePosition[1] {
			slots = []int{0}
		} else if handler.CanPlaceFurnaceFuelSlot(item.ItemID) {
			slots = []int{1}
		} else {
			return false
		}
	}
	limit := player.MaxStackSize(item.ItemID)
	for _, slot := range slots {
		if destination[slot].SameItem(item) && destination[slot].Count < limit {
			destination[slot].Count++
			return true
		}
	}
	for _, slot := range slots {
		if destination[slot].IsEmpty() {
			destination[slot] = player.ItemStack{ItemID: item.ItemID, Count: 1, Damage: item.Damage, Enchantments: item.Enchantments, PotDecorations: item.PotDecorations}
			return true
		}
	}
	return false
}

func insertAutomationStack(kind string, destination []player.ItemStack, item player.ItemStack, destinationPosition, sourcePosition [3]int) bool {
	if item.IsEmpty() {
		return false
	}
	updated := append([]player.ItemStack(nil), destination...)
	for remaining := item.Count; remaining > 0; remaining-- {
		if !insertOneAutomationItem(kind, updated, item, destinationPosition, sourcePosition) {
			return false
		}
	}
	copy(destination, updated)
	return true
}

func containerFacingOffset(facing string) (x, y, z int) {
	switch facing {
	case "up":
		return 0, 1, 0
	case "north":
		return 0, 0, -1
	case "south":
		return 0, 0, 1
	case "west":
		return -1, 0, 0
	case "east":
		return 1, 0, 0
	default:
		return 0, -1, 0
	}
}

func (s *Server) syncAutomatedContainers(world *coreworld.World, dimension int32, positions ...[3]int) {
	if s.game == nil {
		return
	}
	changed := make(map[spatial.BlockPos]struct{}, len(positions))
	for _, position := range positions {
		changed[spatial.BlockPos{X: int32(position[0]), Y: int32(position[1]), Z: int32(position[2])}] = struct{}{}
	}
	s.game.OnlinePlayers(func(p *player.Player) {
		if p.Dimension != dimension || p.OpenContainerKind == "" {
			return
		}
		_, direct := changed[p.OpenContainerPos]
		_, partner := changed[p.OpenContainerPartnerPos]
		if !direct && !partner {
			return
		}
		if p.Edition == player.ClientEditionJava {
			if current, ok := s.sessions.Get(p.UUID); ok {
				handler.RefreshOpenStorageContainer(p, current.Conn, world)
			}
			return
		}
		if isBedrockGenericContainer(p.OpenContainerKind) {
			s.openBedrockGenericContainer(p, p.OpenContainerPos, p.OpenContainerKind)
			if s.bedrockListener != nil {
				s.bedrockListener.SyncGenericContainer(p)
			}
		}
	})
}

func (s *Server) activateDropperOrDispenser(world *coreworld.World, dimension int32, x, y, z int, kind string, changes *[]coreworld.BlockChange) {
	position := [3]int{x, y, z}
	_, stacks, ok := loadAutomationContainer(world, position)
	if !ok {
		return
	}
	slot := -1
	for index, stack := range stacks {
		if !stack.IsEmpty() {
			slot = index
			break
		}
	}
	if slot < 0 {
		handler.BroadcastSoundAt(s.javaSessionsForDimension(dimension), "minecraft:block.dispenser.fail", handler.SoundCategoryBlocks,
			float64(x)+0.5, float64(y)+0.5, float64(z)+0.5, 1, 1)
		return
	}
	block := world.GetBlock(x, y, z)
	dx, dy, dz := containerFacingOffset(block.Properties["facing"])
	target := [3]int{x + dx, y + dy, z + dz}
	item := stacks[slot]

	if kind == "minecraft:dropper" {
		if destinationKind, destination, found := loadAutomationContainer(world, target); found &&
			insertOneAutomationItem(destinationKind, destination, item, target, position) {
			stacks[slot].Count--
			if stacks[slot].Count <= 0 {
				stacks[slot] = player.ItemStack{}
			}
			saveAutomationContainer(world, target, destinationKind, destination)
			saveAutomationContainer(world, position, kind, stacks)
			s.syncAutomatedContainers(world, dimension, position, target)
			return
		}
	}

	handled, remainder := false, player.ItemStack{}
	if kind == "minecraft:dispenser" {
		handled, remainder = s.dispenseSpecialItem(world, dimension, target, item, dx, dy, dz, changes)
	}
	if handled {
		stacks[slot].Count--
	} else {
		s.dispenseDroppedItem(world, item, target, dx, dy, dz)
		stacks[slot].Count--
	}
	if stacks[slot].Count <= 0 {
		stacks[slot] = remainder
	} else if !remainder.IsEmpty() {
		if !insertOneAutomationItem(kind, stacks, remainder, position, target) {
			s.dispenseDroppedItem(world, remainder, target, dx, dy, dz)
		}
	}
	saveAutomationContainer(world, position, kind, stacks)
	s.syncAutomatedContainers(world, dimension, position)
	handler.BroadcastSoundAt(s.javaSessionsForDimension(dimension), "minecraft:block.dispenser.dispense", handler.SoundCategoryBlocks,
		float64(x)+0.5, float64(y)+0.5, float64(z)+0.5, 1, 1)
}

func (s *Server) activateCrafter(world *coreworld.World, dimension int32, x, y, z int) {
	position := [3]int{x, y, z}
	kind, stacks, ok := loadAutomationContainer(world, position)
	if !ok || kind != "minecraft:crafter" || len(stacks) != 9 {
		return
	}
	// Load the disabled-slot bitmask from the block entity metadata slot.
	var disabledSlots uint16
	for _, item := range world.ContainerItems(x, y, z) {
		if item.Slot == handler.CrafterDisabledSlotMetaIndex {
			disabledSlots = uint16(item.Damage)
			break
		}
	}
	var grid [9]player.ItemStack
	for i, stack := range stacks {
		if disabledSlots>>uint(i)&1 == 0 {
			grid[i] = stack
		}
	}
	result := handler.FindCraftingTableResult(grid)
	if result.IsEmpty() {
		handler.BroadcastSoundAt(s.javaSessionsForDimension(dimension), "minecraft:block.dispenser.fail", handler.SoundCategoryBlocks,
			float64(x)+0.5, float64(y)+0.5, float64(z)+0.5, 1, 1)
		return
	}

	block := world.GetBlock(x, y, z)
	facing := strings.SplitN(block.Properties["orientation"], "_", 2)[0]
	dx, dy, dz := containerFacingOffset(facing)
	target := [3]int{x + dx, y + dy, z + dz}
	inserted := false
	if destinationKind, destination, found := loadAutomationContainer(world, target); found &&
		insertAutomationStack(destinationKind, destination, result, target, position) {
		saveAutomationContainer(world, target, destinationKind, destination)
		inserted = true
	}
	if !inserted {
		s.dispenseAutomationStack(world, dimension, result, target, dx, dy, dz)
	}
	for index := range stacks {
		if stacks[index].IsEmpty() {
			continue
		}
		stacks[index].Count--
		if stacks[index].Count <= 0 {
			stacks[index] = player.ItemStack{}
		}
	}
	saveAutomationContainer(world, position, kind, stacks)
	s.syncAutomatedContainers(world, dimension, position, target)
	handler.BroadcastSoundAt(s.javaSessionsForDimension(dimension), "minecraft:block.crafter.craft", handler.SoundCategoryBlocks,
		float64(x)+0.5, float64(y)+0.5, float64(z)+0.5, 1, 1)
}

func (s *Server) dispenseAutomationStack(world *coreworld.World, dimension int32, stack player.ItemStack, target [3]int, dx, dy, dz int) {
	position := spatial.Vec3{X: float64(target[0]) + 0.5, Y: float64(target[1]) + 0.25, Z: float64(target[2]) + 0.5}
	dropped := s.newDroppedItemInWorld(world, stack, position, 0)
	if dropped == nil {
		return
	}
	dropped.VX, dropped.VY, dropped.VZ = float64(dx)*0.2, 0.2+float64(dy)*0.2, float64(dz)*0.2
	handler.BroadcastSpawnMob(dropped, s.javaSessionsForDimension(dimension))
}

func (s *Server) dispenseSpecialItem(world *coreworld.World, dimension int32, target [3]int, item player.ItemStack, dx, dy, dz int, changes *[]coreworld.BlockChange) (bool, player.ItemStack) {
	projectileType := corentity.EntityType("")
	damage := float32(0)
	switch item.ItemID {
	case "minecraft:arrow":
		projectileType, damage = corentity.TypeArrow, 4
	case "minecraft:spectral_arrow":
		projectileType, damage = corentity.TypeSpectralArrow, 4
	case "minecraft:trident":
		projectileType, damage = corentity.TypeTrident, 8
	case "minecraft:wind_charge":
		projectileType = corentity.TypeWindCharge
	case "minecraft:snowball":
		projectileType = corentity.TypeSnowball
	case "minecraft:egg":
		projectileType = corentity.TypeEgg
	case "minecraft:ender_pearl":
		projectileType = corentity.TypeEnderPearl
	case "minecraft:experience_bottle":
		projectileType = corentity.TypeExperienceBottle
	case "minecraft:splash_potion", "minecraft:lingering_potion":
		projectileType = corentity.TypePotion
	case "minecraft:fire_charge":
		projectileType, damage = corentity.TypeSmallFireball, 5
	case "minecraft:tnt":
		if s.game == nil {
			return false, player.ItemStack{}
		}
		entity := corentity.New(s.game.NextEntityID(), newRandomUUID(), corentity.TypePrimedTNT,
			float64(target[0])+0.5, float64(target[1]), float64(target[2])+0.5)
		entity.FuseTicks = 80
		world.Entities.Add(entity)
		handler.BroadcastSpawnMob(entity, s.javaSessionsForDimension(dimension))
		return true, player.ItemStack{}
	case "minecraft:water_bucket", "minecraft:lava_bucket":
		current := world.GetBlock(target[0], target[1], target[2])
		if !current.IsAir() && !coreworld.IsFluidBlock(current.ResourceLocation()) {
			return false, player.ItemStack{}
		}
		fluidName := "minecraft:" + strings.TrimSuffix(strings.TrimPrefix(item.ItemID, "minecraft:"), "_bucket")
		fluid := coreworld.MakeFluid(fluidName, 0)
		world.SetBlock(target[0], target[1], target[2], fluid)
		*changes = append(*changes, coreworld.BlockChange{X: target[0], Y: target[1], Z: target[2], Block: fluid})
		return true, player.ItemStack{ItemID: "minecraft:bucket", Count: 1}
	case "minecraft:bucket":
		current := world.GetBlock(target[0], target[1], target[2])
		if coreworld.FluidLevel(current) != 0 || (current.ResourceLocation() != "minecraft:water" && current.ResourceLocation() != "minecraft:lava") {
			return false, player.ItemStack{}
		}
		filled := strings.TrimPrefix(current.ResourceLocation(), "minecraft:") + "_bucket"
		world.SetBlock(target[0], target[1], target[2], coreworld.Air)
		*changes = append(*changes, coreworld.BlockChange{X: target[0], Y: target[1], Z: target[2], Block: coreworld.Air})
		return true, player.ItemStack{ItemID: "minecraft:" + filled, Count: 1}
	case "minecraft:bone_meal":
		seed := uint64(s.worldAge) + uint64(target[0]*31+target[1]*17+target[2]*13)
		grown, used := world.ApplyBoneMeal(target[0], target[1], target[2], seed)
		if !used {
			return false, player.ItemStack{}
		}
		*changes = append(*changes, grown...)
		return true, player.ItemStack{}
	case "minecraft:flint_and_steel":
		current := world.GetBlock(target[0], target[1], target[2])
		if !current.IsAir() || !coreworld.IsEntitySupportBlock(world.GetBlock(target[0], target[1]-1, target[2]).ResourceLocation()) {
			return false, player.ItemStack{}
		}
		fire := coreworld.Block{Namespace: "minecraft", Name: "fire", Properties: map[string]string{"age": "0"}}
		world.SetBlock(target[0], target[1], target[2], fire)
		world.BlockPhysics.ScheduleFire(target[0], target[1], target[2], s.worldAge, 20)
		*changes = append(*changes, coreworld.BlockChange{X: target[0], Y: target[1], Z: target[2], Block: fire})
		damaged := player.ItemStack{ItemID: item.ItemID, Count: 1, Damage: item.Damage + 1}
		if maximum := player.MaxDurability(item.ItemID); maximum > 0 && damaged.Damage >= maximum {
			damaged = player.ItemStack{}
		}
		return true, damaged
	}
	if strings.HasSuffix(item.ItemID, "_spawn_egg") && s.game != nil {
		entityType := corentity.EntityType(strings.TrimSuffix(item.ItemID, "_spawn_egg"))
		spawned := corentity.New(s.game.NextEntityID(), newRandomUUID(), entityType,
			float64(target[0])+0.5, float64(target[1]), float64(target[2])+0.5)
		world.Entities.Add(spawned)
		handler.BroadcastSpawnMob(spawned, s.javaSessionsForDimension(dimension))
		return true, player.ItemStack{}
	}
	if projectileType == "" || s.game == nil {
		return false, player.ItemStack{}
	}
	entity := corentity.New(s.game.NextEntityID(), newRandomUUID(), projectileType,
		float64(target[0])+0.5, float64(target[1])+0.5, float64(target[2])+0.5)
	entity.ProjectileDamage = damage
	entity.VX, entity.VY, entity.VZ = float64(dx)*1.1, float64(dy)*1.1, float64(dz)*1.1
	if projectileType == corentity.TypeWindCharge {
		entity.VX, entity.VY, entity.VZ = float64(dx)*1.5, float64(dy)*1.5, float64(dz)*1.5
	} else if projectileType == corentity.TypeExperienceBottle || projectileType == corentity.TypePotion {
		entity.VX, entity.VY, entity.VZ = float64(dx)*0.7, float64(dy)*0.7+0.1, float64(dz)*0.7
	} else if projectileType == corentity.TypeSmallFireball {
		entity.VX, entity.VY, entity.VZ = float64(dx)*0.9, float64(dy)*0.9, float64(dz)*0.9
	}
	world.Entities.Add(entity)
	handler.BroadcastSpawnMob(entity, s.javaSessionsForDimension(dimension))
	return true, player.ItemStack{}
}

func (s *Server) dispenseDroppedItem(world *coreworld.World, stack player.ItemStack, target [3]int, dx, dy, dz int) {
	if s.game == nil {
		return
	}
	position := spatial.Vec3{X: float64(target[0]) + 0.5, Y: float64(target[1]) + 0.25, Z: float64(target[2]) + 0.5}
	dropped := s.newDroppedItemInWorld(world, player.ItemStack{ItemID: stack.ItemID, Count: 1, Damage: stack.Damage}, position, 0)
	if dropped == nil {
		return
	}
	dropped.VX, dropped.VY, dropped.VZ = float64(dx)*0.2, 0.2+float64(dy)*0.2, float64(dz)*0.2
	if math.Abs(dropped.VX)+math.Abs(dropped.VZ) == 0 {
		dropped.VZ = 0.02
	}
	handler.BroadcastSpawnMob(dropped, s.javaSessionsForDimension(s.dimensionForWorld(world)))
}

func (s *Server) dimensionForWorld(world *coreworld.World) int32 {
	switch world {
	case s.netherWorld:
		return dimensionNether
	case s.endWorld:
		return dimensionEnd
	default:
		return dimensionOverworld
	}
}
