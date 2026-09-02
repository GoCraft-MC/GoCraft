package handler

import (
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

const (
	chestContainerID     int32 = 1
	chestSlotCount             = 27 // single chest
	doubleChestSlotCount       = 54 // double chest
	doubleChestMenuType  int32 = 5  // minecraft:generic_9x6

	// CrafterDisabledSlotMetaIndex is a sentinel slot index stored in the
	// block entity to persist the 9-bit disabled-slot bitmask. It is never
	// a real inventory slot (crafter has slots 0-8 only).
	CrafterDisabledSlotMetaIndex = 255
)

// openChest opens a single or double chest at pos.  For double chests the
// right half occupies GUI slots 0-26 and the left half 27-53, matching the
// vanilla layout regardless of which half the player clicked.
func openChest(p *player.Player, conn *network.ClientConn, w *coreworld.World, pos spatial.BlockPos) error {
	if p.OpenContainerKind == "minecraft:crafting_table" {
		returnCraftingGrid(p)
	}
	LoadChestContainerState(p, w, pos)

	persistChestContents(p, w)
	menuType := containerMenuType("minecraft:chest") // generic_9x3 = 2
	title := "Chest"
	if p.OpenContainerHasPartner {
		menuType = doubleChestMenuType
		title = "Large Chest"
	}
	if conn == nil {
		return nil
	}
	if err := sendOpenScreen(conn, chestContainerID, menuType, title); err != nil {
		return err
	}
	return sendChestContainerContent(conn, p)
}

func isJavaStorageContainer(kind string) bool {
	switch kind {
	case "minecraft:chest", "minecraft:trapped_chest", "minecraft:barrel",
		"minecraft:hopper", "minecraft:dispenser", "minecraft:dropper", "minecraft:crafter",
		"minecraft:ender_chest",
		"minecraft:shulker_box",
		"minecraft:white_shulker_box", "minecraft:orange_shulker_box",
		"minecraft:magenta_shulker_box", "minecraft:light_blue_shulker_box",
		"minecraft:yellow_shulker_box", "minecraft:lime_shulker_box",
		"minecraft:pink_shulker_box", "minecraft:gray_shulker_box",
		"minecraft:light_gray_shulker_box", "minecraft:cyan_shulker_box",
		"minecraft:purple_shulker_box", "minecraft:blue_shulker_box",
		"minecraft:brown_shulker_box", "minecraft:green_shulker_box",
		"minecraft:red_shulker_box", "minecraft:black_shulker_box":
		return true
	default:
		return false
	}
}

func isShulkerBox(kind string) bool {
	switch kind {
	case "minecraft:shulker_box",
		"minecraft:white_shulker_box", "minecraft:orange_shulker_box",
		"minecraft:magenta_shulker_box", "minecraft:light_blue_shulker_box",
		"minecraft:yellow_shulker_box", "minecraft:lime_shulker_box",
		"minecraft:pink_shulker_box", "minecraft:gray_shulker_box",
		"minecraft:light_gray_shulker_box", "minecraft:cyan_shulker_box",
		"minecraft:purple_shulker_box", "minecraft:blue_shulker_box",
		"minecraft:brown_shulker_box", "minecraft:green_shulker_box",
		"minecraft:red_shulker_box", "minecraft:black_shulker_box":
		return true
	default:
		return false
	}
}

func javaStorageContainerSize(kind string) int {
	switch kind {
	case "minecraft:hopper":
		return 5
	case "minecraft:dispenser", "minecraft:dropper", "minecraft:crafter":
		return 9
	default:
		return chestSlotCount
	}
}

// openStorageContainer connects the Java screen to canonical block-entity
// storage. Chests retain their double-chest rules; other storage blocks use a
// single fixed-size slot array.
func openStorageContainer(p *player.Player, conn *network.ClientConn, w *coreworld.World, pos spatial.BlockPos, kind string) error {
	if kind == "minecraft:chest" || kind == "minecraft:trapped_chest" {
		return openChest(p, conn, w, pos)
	}
	if p.OpenContainerKind == "minecraft:crafting_table" {
		returnCraftingGrid(p)
	}
	p.OpenContainerID = chestContainerID
	p.OpenContainerKind = kind
	p.OpenContainerPos = pos
	p.OpenContainerPartnerPos = spatial.BlockPos{}
	p.OpenContainerHasPartner = false
	p.ContainerSlots = make([]player.ItemStack, javaStorageContainerSize(kind))
	p.CrafterDisabledSlots = 0
	if kind == "minecraft:ender_chest" {
		copy(p.ContainerSlots, p.EnderChestInventory[:])
	} else {
		for _, item := range w.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z)) {
			if item.Slot == CrafterDisabledSlotMetaIndex {
				p.CrafterDisabledSlots = uint16(item.Damage)
				continue
			}
			if item.Slot >= 0 && item.Slot < len(p.ContainerSlots) && item.ItemID != "" && item.Count > 0 {
				p.ContainerSlots[item.Slot] = item.Stack()
			}
		}
	}
	p.ContainerStateID++
	if conn == nil {
		return nil
	}
	if err := sendOpenScreen(conn, chestContainerID, containerMenuType(kind), containerTitle(kind)); err != nil {
		return err
	}
	if err := sendChestContainerContent(conn, p); err != nil {
		return err
	}
	if kind == "minecraft:crafter" {
		return sendCrafterSlotStates(conn, p)
	}
	return nil
}

// RefreshOpenStorageContainer reloads an already-open Java storage screen
// after hopper/dropper automation changes its canonical block entity.
func RefreshOpenStorageContainer(p *player.Player, conn *network.ClientConn, w *coreworld.World) {
	if p == nil || w == nil || !isJavaStorageContainer(p.OpenContainerKind) || p.OpenContainerKind == "minecraft:ender_chest" {
		return
	}
	kind, position := p.OpenContainerKind, p.OpenContainerPos
	if kind == "minecraft:chest" || kind == "minecraft:trapped_chest" {
		LoadChestContainerState(p, w, position)
	} else {
		p.ContainerSlots = make([]player.ItemStack, javaStorageContainerSize(kind))
		for _, item := range w.ContainerItems(int(position.X), int(position.Y), int(position.Z)) {
			if item.Slot >= 0 && item.Slot < len(p.ContainerSlots) && item.ItemID != "" && item.Count > 0 {
				p.ContainerSlots[item.Slot] = item.Stack()
			}
		}
		p.ContainerStateID++
	}
	if conn != nil {
		_ = sendChestContainerContent(conn, p)
	}
}

// LoadChestContainerState loads a single or double chest into the canonical
// open-container fields without sending edition-specific packets. Bedrock and
// Java therefore use exactly the same half ordering and persistence rules.
func LoadChestContainerState(p *player.Player, w *coreworld.World, pos spatial.BlockPos) {
	if p == nil || w == nil {
		return
	}

	block := w.GetBlock(int(pos.X), int(pos.Y), int(pos.Z))
	rightPos, leftPos, hasPartner := chestDoubleHalves(block, pos, w)

	slotCount := chestSlotCount
	if hasPartner {
		slotCount = doubleChestSlotCount
	}

	p.OpenContainerID = chestContainerID
	p.OpenContainerKind = block.ResourceLocation()
	p.OpenContainerPos = rightPos
	p.OpenContainerPartnerPos = leftPos
	p.OpenContainerHasPartner = hasPartner
	p.ContainerSlots = make([]player.ItemStack, slotCount)

	// Right half → slots 0-26.
	for _, item := range w.ContainerItems(int(rightPos.X), int(rightPos.Y), int(rightPos.Z)) {
		if item.Slot >= 0 && item.Slot < chestSlotCount && item.ItemID != "" && item.Count > 0 {
			p.ContainerSlots[item.Slot] = item.Stack()
		}
	}
	// Left half → slots 27-53.
	if hasPartner {
		for _, item := range w.ContainerItems(int(leftPos.X), int(leftPos.Y), int(leftPos.Z)) {
			slot := item.Slot + chestSlotCount
			if item.Slot >= 0 && item.Slot < chestSlotCount && slot < slotCount && item.ItemID != "" && item.Count > 0 {
				p.ContainerSlots[slot] = item.Stack()
			}
		}
	}

	p.ContainerStateID++
}

// PersistChestContents stores the canonical open chest back into one or both
// block entities. It is exported for the Bedrock adapter's shared simulation.
func PersistChestContents(p *player.Player, w *coreworld.World) { persistChestContents(p, w) }

// chestDoubleHalves resolves the right-half and left-half positions for a
// chest block.  If the block is a single chest, both returned positions equal
// pos and hasPartner is false.  The right half always receives GUI slots 0-26.
func chestDoubleHalves(block coreworld.Block, pos spatial.BlockPos, w *coreworld.World) (rightPos, leftPos spatial.BlockPos, hasPartner bool) {
	chestType := block.Properties["type"]
	facing := block.Properties["facing"]
	if chestType == "" || chestType == "single" || facing == "" {
		return pos, pos, false
	}

	rp, lp := chestSidePositions(facing, pos)

	var partnerPos spatial.BlockPos
	if chestType == "right" {
		// This IS the right half; partner is the left half.
		partnerPos = lp
		rightPos = pos
	} else {
		// This IS the left half; partner is the right half.
		partnerPos = rp
		rightPos = partnerPos
		leftPos = pos
	}

	partner := w.GetBlock(int(partnerPos.X), int(partnerPos.Y), int(partnerPos.Z))
	if partner.ResourceLocation() != block.ResourceLocation() {
		return pos, pos, false
	}
	if chestType == "right" {
		leftPos = partnerPos
	}
	return rightPos, leftPos, true
}

// placeChestBlock places a chest at (px, py, pz) with the correct facing and
// type properties, linking it to any adjacent single chest of the same kind.
// It calls applyBlockChange for the new block and, when a pair forms, for the
// updated neighbour as well.
func placeChestBlock(p *player.Player, px, py, pz int, kind string, w *coreworld.World, mgr *session.Manager) {
	facing := chestFacingFromYaw(p.Rotation.Yaw)
	rPos, lPos := chestSidePositions(facing, spatial.BlockPos{X: int32(px), Y: int32(py), Z: int32(pz)})

	// Helper: check if an adjacent block is a linkable single chest of the same kind.
	isSingle := func(candidate spatial.BlockPos) (coreworld.Block, bool) {
		b := w.GetBlock(int(candidate.X), int(candidate.Y), int(candidate.Z))
		if b.ResourceLocation() != kind {
			return coreworld.Block{}, false
		}
		t := b.Properties["type"]
		return b, t == "" || t == "single"
	}

	var newType, neighborType string
	var neighborPos spatial.BlockPos
	var neighborBlock coreworld.Block

	if nb, ok := isSingle(rPos); ok {
		// Existing chest is at the RIGHT position → it stays type=right, new is type=left.
		newType, neighborType = "left", "right"
		neighborPos, neighborBlock = rPos, nb
	} else if nb, ok := isSingle(lPos); ok {
		// Existing chest is at the LEFT position → it stays type=left, new is type=right.
		newType, neighborType = "right", "left"
		neighborPos, neighborBlock = lPos, nb
	}

	newBlock := chestBlockWithProps(kind, facing, chestTypeOrSingle(newType))
	applyBlockChange(px, py, pz, newBlock, w, mgr)

	if neighborType != "" {
		updatedNeighbor := chestBlockWithProps(kind, chestFacingFromBlock(neighborBlock, facing), neighborType)
		applyBlockChange(int(neighborPos.X), int(neighborPos.Y), int(neighborPos.Z), updatedNeighbor, w, mgr)
	}
}

// unlinkChestPartner resets the partner of a broken double-chest half back to
// type=single so it renders correctly as a standalone chest.
func unlinkChestPartner(x, y, z int, broken coreworld.Block, w *coreworld.World, mgr *session.Manager) {
	kind := broken.ResourceLocation()
	if kind != "minecraft:chest" && kind != "minecraft:trapped_chest" {
		return
	}
	chestType := broken.Properties["type"]
	facing := broken.Properties["facing"]
	if chestType == "" || chestType == "single" || facing == "" {
		return
	}
	pos := spatial.BlockPos{X: int32(x), Y: int32(y), Z: int32(z)}
	rPos, lPos := chestSidePositions(facing, pos)
	var partnerPos spatial.BlockPos
	if chestType == "right" {
		partnerPos = lPos
	} else {
		partnerPos = rPos
	}
	partner := w.GetBlock(int(partnerPos.X), int(partnerPos.Y), int(partnerPos.Z))
	if partner.ResourceLocation() != kind {
		return
	}
	reset := chestBlockWithProps(kind, chestFacingFromBlock(partner, facing), "single")
	applyBlockChange(int(partnerPos.X), int(partnerPos.Y), int(partnerPos.Z), reset, w, mgr)
}

// chestFacingFromYaw returns the facing direction for a chest placed by a
// player with the given yaw.  Chests face TOWARD the player who placed them.
func chestFacingFromYaw(yaw float32) string {
	// Minecraft yaw: 0=south, 90=west, ±180=north, -90=east.
	y := yaw
	for y > 180 {
		y -= 360
	}
	for y < -180 {
		y += 360
	}
	switch {
	case y >= -45 && y < 45:
		return "north" // player faces south → chest front faces north (toward player)
	case y >= 45 && y < 135:
		return "east" // player faces west → chest faces east
	case y >= -135 && y < -45:
		return "west" // player faces east → chest faces west
	default:
		return "south" // player faces north → chest faces south
	}
}

// chestFacingFromBlock returns the facing stored in an existing chest block's
// properties, falling back to the given default.
func chestFacingFromBlock(b coreworld.Block, fallback string) string {
	if f := b.Properties["facing"]; f != "" {
		return f
	}
	return fallback
}

// chestSidePositions returns the (right, left) neighbour positions for a chest
// with the given facing, where right/left are relative to the player standing
// in front of the chest and looking at it.
//
//	facing=north → player at south looking north: right=east (+X), left=west (-X)
//	facing=south → player at north looking south: right=west (-X), left=east (+X)
//	facing=east  → player at west looking east:  right=south (+Z), left=north (-Z)
//	facing=west  → player at east looking west:  right=north (-Z), left=south (+Z)
func chestSidePositions(facing string, pos spatial.BlockPos) (rightPos, leftPos spatial.BlockPos) {
	switch facing {
	case "north":
		return spatial.BlockPos{X: pos.X + 1, Y: pos.Y, Z: pos.Z},
			spatial.BlockPos{X: pos.X - 1, Y: pos.Y, Z: pos.Z}
	case "south":
		return spatial.BlockPos{X: pos.X - 1, Y: pos.Y, Z: pos.Z},
			spatial.BlockPos{X: pos.X + 1, Y: pos.Y, Z: pos.Z}
	case "east":
		return spatial.BlockPos{X: pos.X, Y: pos.Y, Z: pos.Z + 1},
			spatial.BlockPos{X: pos.X, Y: pos.Y, Z: pos.Z - 1}
	default: // west
		return spatial.BlockPos{X: pos.X, Y: pos.Y, Z: pos.Z - 1},
			spatial.BlockPos{X: pos.X, Y: pos.Y, Z: pos.Z + 1}
	}
}

func chestBlockWithProps(kind, facing, chestType string) coreworld.Block {
	ns, name := "minecraft", kind
	if len(kind) > 10 && kind[:10] == "minecraft:" {
		name = kind[10:]
	}
	return coreworld.Block{
		Namespace: ns,
		Name:      name,
		Properties: map[string]string{
			"facing":      facing,
			"type":        chestType,
			"waterlogged": "false",
		},
	}
}

func chestTypeOrSingle(t string) string {
	if t == "" {
		return "single"
	}
	return t
}

func sendChestContainerContent(conn *network.ClientConn, p *player.Player) error {
	slotCount := len(p.ContainerSlots)
	b := protocol.NewBuilder(packetIDSetContainerContent).
		VarInt(chestContainerID).
		VarInt(p.ContainerStateID).
		VarInt(int32(slotCount + 36))
	for i := 0; i < slotCount; i++ {
		if i < len(p.ContainerSlots) {
			encodeSlot(b, p.ContainerSlots[i])
		} else {
			encodeSlot(b, player.ItemStack{})
		}
	}
	for i := 9; i < player.HotbarStart; i++ {
		encodeSlot(b, p.Inventory[i])
	}
	for i := player.HotbarStart; i < player.HotbarStart+9; i++ {
		encodeSlot(b, p.Inventory[i])
	}
	encodeSlot(b, p.CarriedItem)
	return conn.WritePacket(b.Build())
}

func handleChestClick(p *player.Player, w *coreworld.World, slot int, button byte, mode int32) {
	switch mode {
	case 0:
		clickChestSlot(p, slot, button)
	case 1:
		shiftChestSlot(p, slot)
	}
	p.ContainerStateID++
	persistStorageContents(p, w)
}

func persistStorageContents(p *player.Player, w *coreworld.World) {
	if p == nil || w == nil || !isJavaStorageContainer(p.OpenContainerKind) {
		return
	}
	if p.OpenContainerKind == "minecraft:chest" || p.OpenContainerKind == "minecraft:trapped_chest" {
		persistChestContents(p, w)
		return
	}
	if p.OpenContainerKind == "minecraft:ender_chest" {
		clear(p.EnderChestInventory[:])
		copy(p.EnderChestInventory[:], p.ContainerSlots)
		return
	}
	items := make([]coreworld.ContainerItem, 0, len(p.ContainerSlots)+1)
	for slot, stack := range p.ContainerSlots {
		if !stack.IsEmpty() {
			items = append(items, coreworld.ContainerItemFromStack(slot, stack))
		}
	}
	if p.OpenContainerKind == "minecraft:crafter" && p.CrafterDisabledSlots != 0 {
		items = append(items, coreworld.ContainerItem{Slot: CrafterDisabledSlotMetaIndex, Damage: int(p.CrafterDisabledSlots)})
	}
	pos := p.OpenContainerPos
	w.SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), p.OpenContainerKind, items)
}

func clickChestSlot(p *player.Player, containerSlot int, button byte) {
	if containerSlot == -999 {
		if button == 0 {
			p.CarriedItem = player.ItemStack{}
		} else if !p.CarriedItem.IsEmpty() {
			p.CarriedItem.Count--
			normalizeStack(&p.CarriedItem)
		}
		return
	}
	target := chestContainerSlot(p, containerSlot)
	if target == nil {
		return
	}
	if button == 0 {
		switch {
		case p.CarriedItem.IsEmpty():
			p.CarriedItem, *target = *target, player.ItemStack{}
		case target.IsEmpty():
			*target, p.CarriedItem = p.CarriedItem, player.ItemStack{}
		case target.SameItem(p.CarriedItem) && target.Count < 64:
			add := minInt(64-target.Count, p.CarriedItem.Count)
			target.Count += add
			p.CarriedItem.Count -= add
			normalizeStack(&p.CarriedItem)
		default:
			p.CarriedItem, *target = *target, p.CarriedItem
		}
		return
	}
	if p.CarriedItem.IsEmpty() {
		if target.IsEmpty() {
			return
		}
		take := (target.Count + 1) / 2
		p.CarriedItem = *target
		p.CarriedItem.Count = take
		target.Count -= take
		normalizeStack(target)
	} else if target.IsEmpty() {
		*target = p.CarriedItem
		target.Count = 1
		p.CarriedItem.Count--
		normalizeStack(&p.CarriedItem)
	} else if target.SameItem(p.CarriedItem) && target.Count < 64 {
		target.Count++
		p.CarriedItem.Count--
		normalizeStack(&p.CarriedItem)
	}
}

func shiftChestSlot(p *player.Player, containerSlot int) {
	slotCount := len(p.ContainerSlots)
	target := chestContainerSlot(p, containerSlot)
	if target == nil || target.IsEmpty() {
		return
	}
	if containerSlot < slotCount {
		inventory := p.Inventory
		if addStackToInventory(&inventory, *target) {
			p.Inventory = inventory
			*target = player.ItemStack{}
		}
		return
	}
	slots := append([]player.ItemStack(nil), p.ContainerSlots...)
	if addStackToContainer(slots, *target) {
		p.ContainerSlots = slots
		*target = player.ItemStack{}
	}
}

func addStackToContainer(slots []player.ItemStack, item player.ItemStack) bool {
	if item.IsEmpty() {
		return true
	}
	capacity := 0
	for _, slot := range slots {
		switch {
		case slot.IsEmpty():
			capacity += 64
		case slot.SameItem(item) && slot.Count < 64:
			capacity += 64 - slot.Count
		}
	}
	if capacity < item.Count {
		return false
	}
	remaining := item.Count
	for i := range slots {
		if !slots[i].SameItem(item) || slots[i].Count >= 64 {
			continue
		}
		add := minInt(64-slots[i].Count, remaining)
		slots[i].Count += add
		remaining -= add
		if remaining == 0 {
			return true
		}
	}
	for i := range slots {
		if !slots[i].IsEmpty() {
			continue
		}
		add := minInt(64, remaining)
		slots[i] = item
		slots[i].Count = add
		remaining -= add
		if remaining == 0 {
			return true
		}
	}
	return remaining == 0
}

// chestContainerSlot maps a protocol container-slot index to the underlying
// ItemStack for a single or double chest.  The chest region is [0, slotCount),
// the main inventory is [slotCount, slotCount+27), hotbar is [slotCount+27, slotCount+36).
func chestContainerSlot(p *player.Player, containerSlot int) *player.ItemStack {
	slotCount := len(p.ContainerSlots)
	switch {
	case containerSlot >= 0 && containerSlot < slotCount:
		return &p.ContainerSlots[containerSlot]
	case containerSlot >= slotCount && containerSlot < slotCount+27:
		return &p.Inventory[9+containerSlot-slotCount]
	case containerSlot >= slotCount+27 && containerSlot < slotCount+36:
		return &p.Inventory[player.HotbarStart+containerSlot-(slotCount+27)]
	default:
		return nil
	}
}

// persistChestContents saves the open container's items back to the world.
// For double chests slots 0-26 go to the right half and 27-53 to the left half.
func persistChestContents(p *player.Player, w *coreworld.World) {
	kind := p.OpenContainerKind
	if kind != "minecraft:chest" && kind != "minecraft:trapped_chest" {
		kind = "minecraft:chest"
	}
	pos := p.OpenContainerPos
	if p.OpenContainerHasPartner {
		// Save right half (slots 0-26).
		var rightItems []coreworld.ContainerItem
		for slot := 0; slot < chestSlotCount && slot < len(p.ContainerSlots); slot++ {
			stack := p.ContainerSlots[slot]
			if !stack.IsEmpty() {
				rightItems = append(rightItems, coreworld.ContainerItemFromStack(slot, stack))
			}
		}
		w.SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), kind, rightItems)

		// Save left half (slots 27-53).
		lp := p.OpenContainerPartnerPos
		var leftItems []coreworld.ContainerItem
		for slot := chestSlotCount; slot < doubleChestSlotCount && slot < len(p.ContainerSlots); slot++ {
			stack := p.ContainerSlots[slot]
			if !stack.IsEmpty() {
				leftItems = append(leftItems, coreworld.ContainerItemFromStack(slot-chestSlotCount, stack))
			}
		}
		w.SetContainerItems(int(lp.X), int(lp.Y), int(lp.Z), kind, leftItems)
		return
	}

	// Single chest.
	items := make([]coreworld.ContainerItem, 0, len(p.ContainerSlots))
	for slot, stack := range p.ContainerSlots {
		if stack.IsEmpty() {
			continue
		}
		items = append(items, coreworld.ContainerItemFromStack(slot, stack))
	}
	w.SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), kind, items)
}

// sendCrafterSlotStates sends SetContainerData for each of the 9 crafter
// slots so the client knows which are disabled (value=1) vs enabled (value=0).
func sendCrafterSlotStates(conn *network.ClientConn, p *player.Player) error {
	for slot := 0; slot < 9; slot++ {
		value := int32(0)
		if p.CrafterDisabledSlots>>uint(slot)&1 == 1 {
			value = 1
		}
		pkt := protocol.NewBuilder(packetIDSetContainerData).
			VarInt(int32(chestContainerID)).
			VarInt(int32(slot)).
			VarInt(value).
			Build()
		if err := conn.WritePacket(pkt); err != nil {
			return err
		}
	}
	return nil
}

// handleCrafterButtonClick toggles a crafter slot's disabled state when the
// player right-clicks its slot header. The Java client sends a
// ContainerButtonClick with buttonID = slot index (0-8).
func handleCrafterButtonClick(p *player.Player, conn *network.ClientConn, w *coreworld.World, buttonID int32) error {
	if p == nil || p.OpenContainerKind != "minecraft:crafter" || buttonID < 0 || buttonID >= 9 {
		return nil
	}
	// Toggle the bit.
	p.CrafterDisabledSlots ^= 1 << uint(buttonID)
	// Clear the slot's item if it's now disabled.
	if p.CrafterDisabledSlots>>uint(buttonID)&1 == 1 && int(buttonID) < len(p.ContainerSlots) {
		if !p.ContainerSlots[buttonID].IsEmpty() {
			p.GiveItem(p.ContainerSlots[buttonID])
			p.ContainerSlots[buttonID] = player.ItemStack{}
		}
	}
	persistStorageContents(p, w)
	p.ContainerStateID++
	if conn == nil {
		return nil
	}
	value := int32(0)
	if p.CrafterDisabledSlots>>uint(buttonID)&1 == 1 {
		value = 1
	}
	pkt := protocol.NewBuilder(packetIDSetContainerData).
		VarInt(int32(chestContainerID)).
		VarInt(buttonID).
		VarInt(value).
		Build()
	return conn.WritePacket(pkt)
}
