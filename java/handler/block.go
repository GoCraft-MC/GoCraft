package handler

// Block interaction handling for Milestone 8.
//
// Receives C→S Player Action (digging) and Use Item On (block placement)
// packets, mutates the canonical core/world, and broadcasts Block Update to
// every connected player.
//
// The packet layouts and IDs below target Minecraft Java 1.21.4,
// protocol 769.

import (
	"fmt"
	"log/slog"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	javaworld "GoCraft/java/world"
)

// digBreaksBlock reports whether the given Player Action status should break
// the targeted block for a player in the given game mode.
//
// Creative mode breaks on START_DIGGING (status 0) because the client does
// not run a mining animation — the block disappears immediately.
// Survival mode (and adventure) breaks on FINISH_DIGGING (status 2) after the
// full mining animation completes.
//
// In both cases CANCEL_DIGGING (status 1) is left to the caller; no world
// change is made.
func digBreaksBlock(status int32, mode player.GameMode) bool {
	switch mode {
	case player.GameModeCreative:
		return status == actionStatusStartDigging
	default: // survival, adventure
		return status == actionStatusFinishDigging
	}
}

// Player Action status codes (field "status" in C→S Player Action).
const (
	actionStatusStartDigging  = 0 // block targeted — instant break in creative
	actionStatusCancelDigging = 1 // player looked away / right-clicked before break
	actionStatusFinishDigging = 2 // break animation completed (survival)
)

// ── Dispatch ──────────────────────────────────────────────────────────────────

// handleBlockPacket dispatches an incoming block-interaction packet.
// Called from the play loop for packets that need the world and session manager.
func handleBlockPacket(pkt *protocol.Packet, p *player.Player, w *coreworld.World, mgr *session.Manager, conn *network.ClientConn) error {
	switch pkt.ID {
	case packetIDPlayerAction:
		return handlePlayerAction(pkt, p, w, mgr)
	case packetIDUseItemOn:
		return handleUseItemOn(pkt, p, w, mgr, conn)
	}
	return nil
}

// ── C→S handlers ─────────────────────────────────────────────────────────────

// handlePlayerAction handles C→S Player Action.
//
// Wire layout (1.21.4):
//
//	VarInt    status   (0=start, 1=cancel, 2=finish digging)
//	Long      location (packed block position: X«38 | Z«12 | Y)
//	Byte      face     (0=−Y, 1=+Y, 2=−Z, 3=+Z, 4=−X, 5=+X)
//	VarInt    sequence (monotonic counter; echoed in Acknowledge Block Change)
func handlePlayerAction(pkt *protocol.Packet, p *player.Player, w *coreworld.World, mgr *session.Manager) error {
	r := pkt.Reader()

	status, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("player action: reading status: %w", err)
	}
	bx, by, bz, err := protocol.ReadPosition(r)
	if err != nil {
		return fmt.Errorf("player action: reading position: %w", err)
	}
	if _, err := protocol.ReadByte(r); err != nil { // face — unused in M8
		return fmt.Errorf("player action: reading face: %w", err)
	}
	seq, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("player action: reading sequence: %w", err)
	}

	// Reject out-of-bounds Y before touching the world.
	if int(by) < coreworld.WorldMinY || int(by) > coreworld.WorldMaxY {
		slog.Warn("player action: Y out of bounds", "player", p.Username, "y", by)
		sendAcknowledgeBlockChange(mgr, p, seq)
		return nil
	}

	if digBreaksBlock(status, p.GameMode) {
		broken := w.GetBlock(int(bx), int(by), int(bz))
		slog.Info("block break", "player", p.Username,
			"x", bx, "y", by, "z", bz,
			"block", broken.ResourceLocation(),
			"mode", p.GameMode, "status", status)
		applyBlockChange(int(bx), int(by), int(bz), coreworld.Air, w, mgr)

		// Give drop to player in survival/adventure mode.
		if p.GameMode != player.GameModeCreative && p.GameMode != player.GameModeSpectator {
			dropName, dropCount := blockDropItem(broken.ResourceLocation())
			if dropName != "" && dropCount > 0 {
				if p.GiveItem(player.ItemStack{ItemID: dropName, Count: dropCount}) {
					// Sync updated inventory to the client.
					sess, ok := mgr.Get(p.UUID)
					if ok {
						_ = sendSetContainerContent(sess.Conn, p, 1)
					}
				}
			}
		}
	}

	// Always acknowledge so the client does not roll back its optimistic update.
	sendAcknowledgeBlockChange(mgr, p, seq)
	return nil
}

// faceOffset maps a Use Item On face index to the (dx, dy, dz) offset of the
// block being placed relative to the targeted block.
//
//	0: −Y (bottom face → place below)
//	1: +Y (top face    → place above, most common)
//	2: −Z (north face)
//	3: +Z (south face)
//	4: −X (west face)
//	5: +X (east face)
var faceOffset = [6][3]int32{
	{0, -1, 0},
	{0, +1, 0},
	{0, 0, -1},
	{0, 0, +1},
	{-1, 0, 0},
	{+1, 0, 0},
}

// containerMenuType maps a block resource location to its minecraft:menu
// protocol ID when right-clicking opens a container UI.
// Returns -1 if the block is not an interactive container.
func containerMenuType(blockName string) int32 {
	switch blockName {
	case "minecraft:crafting_table":
		return 12 // minecraft:crafting
	case "minecraft:furnace", "minecraft:lit_furnace":
		return 14 // minecraft:furnace
	case "minecraft:blast_furnace", "minecraft:lit_blast_furnace":
		return 10 // minecraft:blast_furnace
	case "minecraft:smoker", "minecraft:lit_smoker":
		return 22 // minecraft:smoker
	case "minecraft:anvil", "minecraft:chipped_anvil", "minecraft:damaged_anvil":
		return 8 // minecraft:anvil
	case "minecraft:enchanting_table":
		return 13 // minecraft:enchantment
	case "minecraft:grindstone":
		return 15 // minecraft:grindstone
	case "minecraft:loom":
		return 18 // minecraft:loom
	case "minecraft:smithing_table":
		return 21 // minecraft:smithing
	case "minecraft:stonecutter":
		return 24 // minecraft:stonecutter
	case "minecraft:brewing_stand":
		return 11 // minecraft:brewing_stand
	case "minecraft:cartography_table":
		return 23 // minecraft:cartography_table
	case "minecraft:beacon":
		return 9 // minecraft:beacon
	case "minecraft:chest", "minecraft:trapped_chest", "minecraft:barrel",
		"minecraft:ender_chest":
		return 2 // minecraft:generic_9x3
	case "minecraft:hopper":
		return 16 // minecraft:hopper
	case "minecraft:dispenser", "minecraft:dropper":
		return 6 // minecraft:generic_3x3
	case "minecraft:shulker_box",
		"minecraft:white_shulker_box", "minecraft:orange_shulker_box",
		"minecraft:magenta_shulker_box", "minecraft:light_blue_shulker_box",
		"minecraft:yellow_shulker_box", "minecraft:lime_shulker_box",
		"minecraft:pink_shulker_box", "minecraft:gray_shulker_box",
		"minecraft:light_gray_shulker_box", "minecraft:cyan_shulker_box",
		"minecraft:purple_shulker_box", "minecraft:blue_shulker_box",
		"minecraft:brown_shulker_box", "minecraft:green_shulker_box",
		"minecraft:red_shulker_box", "minecraft:black_shulker_box":
		return 20 // minecraft:shulker_box
	}
	return -1
}

// blockDropItem returns the canonical item name and count that should drop when
// the block with the given resource location is broken without silk touch.
// Returns ("", 0) if the block drops nothing.
func blockDropItem(blockName string) (string, int) {
	switch blockName {
	// Stone-family: drop cobblestone
	case "minecraft:stone":
		return "minecraft:cobblestone", 1
	case "minecraft:infested_stone":
		return "", 0 // silverfish block — no drop

	// Grass / dirt variants
	case "minecraft:grass_block", "minecraft:mycelium", "minecraft:podzol":
		return "minecraft:dirt", 1
	case "minecraft:farmland", "minecraft:dirt_path":
		return "minecraft:dirt", 1

	// Coal ore
	case "minecraft:coal_ore", "minecraft:deepslate_coal_ore":
		return "minecraft:coal", 1

	// Iron ore
	case "minecraft:iron_ore", "minecraft:deepslate_iron_ore":
		return "minecraft:raw_iron", 1

	// Gold ore
	case "minecraft:gold_ore", "minecraft:deepslate_gold_ore":
		return "minecraft:raw_gold", 1
	case "minecraft:nether_gold_ore":
		return "minecraft:gold_nugget", 4

	// Diamond ore
	case "minecraft:diamond_ore", "minecraft:deepslate_diamond_ore":
		return "minecraft:diamond", 1

	// Emerald ore
	case "minecraft:emerald_ore", "minecraft:deepslate_emerald_ore":
		return "minecraft:emerald", 1

	// Lapis ore
	case "minecraft:lapis_ore", "minecraft:deepslate_lapis_ore":
		return "minecraft:lapis_lazuli", 4

	// Redstone ore
	case "minecraft:redstone_ore", "minecraft:deepslate_redstone_ore",
		"minecraft:lit_redstone_ore", "minecraft:lit_deepslate_redstone_ore":
		return "minecraft:redstone", 4

	// Copper ore
	case "minecraft:copper_ore", "minecraft:deepslate_copper_ore":
		return "minecraft:raw_copper", 2

	// Nether quartz ore
	case "minecraft:nether_quartz_ore":
		return "minecraft:quartz", 1

	// Ancient debris (drops itself)
	case "minecraft:ancient_debris":
		return "minecraft:ancient_debris", 1

	// Nether wart
	case "minecraft:nether_wart":
		return "minecraft:nether_wart", 2

	// Gravel (simplification: always drops gravel, not flint)
	case "minecraft:gravel":
		return "minecraft:gravel", 1

	// Clay
	case "minecraft:clay":
		return "minecraft:clay_ball", 4

	// Glowstone
	case "minecraft:glowstone":
		return "minecraft:glowstone_dust", 2

	// Sea lantern
	case "minecraft:sea_lantern":
		return "minecraft:prismarine_crystals", 2

	// Leaves — drop nothing without silk touch (simplification: no sapling chance)
	case "minecraft:oak_leaves", "minecraft:birch_leaves", "minecraft:spruce_leaves",
		"minecraft:jungle_leaves", "minecraft:acacia_leaves", "minecraft:dark_oak_leaves",
		"minecraft:cherry_leaves", "minecraft:azalea_leaves", "minecraft:flowering_azalea_leaves",
		"minecraft:mangrove_leaves":
		return "", 0

	// Short/tall plants — drop nothing
	case "minecraft:short_grass", "minecraft:grass", "minecraft:fern",
		"minecraft:tall_grass", "minecraft:large_fern",
		"minecraft:dead_bush", "minecraft:seagrass", "minecraft:tall_seagrass",
		"minecraft:dandelion", "minecraft:poppy", "minecraft:allium",
		"minecraft:azure_bluet", "minecraft:red_tulip", "minecraft:orange_tulip",
		"minecraft:white_tulip", "minecraft:pink_tulip", "minecraft:oxeye_daisy",
		"minecraft:cornflower", "minecraft:lily_of_the_valley", "minecraft:blue_orchid",
		"minecraft:sunflower", "minecraft:lilac", "minecraft:rose_bush", "minecraft:peony",
		"minecraft:wither_rose", "minecraft:torchflower",
		"minecraft:vine", "minecraft:moss_carpet",
		"minecraft:brown_mushroom", "minecraft:red_mushroom":
		return "", 0

	// Air — nothing
	case "", "minecraft:air", "minecraft:cave_air", "minecraft:void_air",
		"minecraft:water", "minecraft:lava":
		return "", 0

	// Everything else drops itself
	default:
		return blockName, 1
	}
}

// handleUseItemOn handles C→S Use Item On (block placement).
//
// Wire layout (1.21.4):
//
//	VarInt    hand         (0=main hand, 1=off hand)
//	Long      location     (packed block position of the targeted block)
//	VarInt    face         (0=−Y, 1=+Y, 2=−Z, 3=+Z, 4=−X, 5=+X)
//	Float     cursor_x/y/z (hit position within the target face, 0.0–1.0)
//	Bool      inside_block (player head is inside a block)
//	Bool      world_border_hit
//	VarInt    sequence
func handleUseItemOn(pkt *protocol.Packet, p *player.Player, w *coreworld.World, mgr *session.Manager, conn *network.ClientConn) error {
	r := pkt.Reader()

	if _, err := protocol.ReadVarInt(r); err != nil { // hand
		return fmt.Errorf("use item on: reading hand: %w", err)
	}
	bx, by, bz, err := protocol.ReadPosition(r)
	if err != nil {
		return fmt.Errorf("use item on: reading position: %w", err)
	}
	face, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("use item on: reading face: %w", err)
	}
	if _, err := protocol.ReadFloat(r); err != nil { // cursor_x
		return fmt.Errorf("use item on: reading cursor_x: %w", err)
	}
	if _, err := protocol.ReadFloat(r); err != nil { // cursor_y
		return fmt.Errorf("use item on: reading cursor_y: %w", err)
	}
	if _, err := protocol.ReadFloat(r); err != nil { // cursor_z
		return fmt.Errorf("use item on: reading cursor_z: %w", err)
	}
	if _, err := protocol.ReadBool(r); err != nil { // inside_block
		return fmt.Errorf("use item on: reading inside_block: %w", err)
	}
	if _, err := protocol.ReadBool(r); err != nil { // world_border_hit
		return fmt.Errorf("use item on: reading world_border_hit: %w", err)
	}
	seq, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("use item on: reading sequence: %w", err)
	}

	// Container blocks: right-clicking opens a UI instead of placing a block.
	// (Sneaking to bypass is not yet tracked; always open the container.)
	targetBlock := w.GetBlock(int(bx), int(by), int(bz))
	if menuType := containerMenuType(targetBlock.ResourceLocation()); menuType >= 0 {
		title := containerTitle(targetBlock.ResourceLocation())
		slog.Info("container opened", "player", p.Username, "block", targetBlock.ResourceLocation())
		sendAcknowledgeBlockChange(mgr, p, seq)
		return sendOpenScreen(conn, 1, menuType, title)
	}

	// Resolve placement position: target block + face offset.
	if face < 0 || int(face) >= len(faceOffset) {
		sendAcknowledgeBlockChange(mgr, p, seq)
		return nil
	}
	off := faceOffset[face]
	px, py, pz := int(bx+off[0]), int(by+off[1]), int(bz+off[2])

	// Bounds-check the placement Y.
	if py < coreworld.WorldMinY || py > coreworld.WorldMaxY {
		sendAcknowledgeBlockChange(mgr, p, seq)
		return nil
	}

	// Resolve the block from the held item.
	held := p.HeldItem()
	if held.IsEmpty() || !javaworld.IsPlaceableAsBlock(held.ItemID) {
		// Nothing to place — acknowledge so the client reverts its preview.
		sendAcknowledgeBlockChange(mgr, p, seq)
		return nil
	}

	// Refuse to overwrite an occupied block.
	if existing := w.GetBlock(px, py, pz); !existing.IsAir() {
		sendAcknowledgeBlockChange(mgr, p, seq)
		return nil
	}

	block := javaworld.ItemIDToBlock(held.ItemID)
	slog.Info("block place", "player", p.Username,
		"block", block.ResourceLocation(), "x", px, "y", py, "z", pz)
	applyBlockChange(px, py, pz, block, w, mgr)
	sendAcknowledgeBlockChange(mgr, p, seq)
	return nil
}

// ── World mutation + broadcast ────────────────────────────────────────────────

// applyBlockChange sets the block at (x, y, z) in the canonical world and
// broadcasts a Block Update packet to every connected player.
func applyBlockChange(x, y, z int, block coreworld.Block, w *coreworld.World, mgr *session.Manager) {
	w.SetBlock(x, y, z, block)
	stateID := javaworld.StateID(block)
	pkt := buildBlockUpdate(x, y, z, stateID)
	for _, s := range mgr.SnapshotAll() {
		_ = s.Conn.WritePacket(pkt)
	}
}

// sendAcknowledgeBlockChange sends an Acknowledge Block Change packet to the
// session identified by p.UUID.
func sendAcknowledgeBlockChange(mgr *session.Manager, p *player.Player, seq int32) {
	sess, ok := mgr.Get(p.UUID)
	if !ok {
		return
	}
	_ = sess.Conn.WritePacket(buildAcknowledgeBlockChange(seq))
}

// ── Packet builders ───────────────────────────────────────────────────────────

// buildBlockUpdate constructs a Block Update (S→C) packet.
//
// Wire layout (1.21.4):
//
//	Long    location (packed: X«38 | Z«12 | Y)
//	VarInt  block_state_id
func buildBlockUpdate(x, y, z int, stateID int32) *protocol.Packet {
	return protocol.NewBuilder(packetIDBlockUpdate).
		Long(packBlockPos(x, y, z)).
		VarInt(stateID).
		Build()
}

// buildAcknowledgeBlockChange constructs an Acknowledge Block Change (S→C) packet.
//
// Wire layout (1.21.4):
//
//	VarInt  sequence_id
func buildAcknowledgeBlockChange(seq int32) *protocol.Packet {
	return protocol.NewBuilder(packetIDAcknowledgeBlockChange).
		VarInt(seq).
		Build()
}

// packBlockPos encodes absolute block coordinates as the Minecraft 64-bit
// packed Position: X(26 bits) | Z(26 bits) | Y(12 bits).
func packBlockPos(x, y, z int) int64 {
	return ((int64(x) & 0x3FFFFFF) << 38) |
		((int64(z) & 0x3FFFFFF) << 12) |
		(int64(y) & 0xFFF)
}

// containerTitle returns a human-readable title for a container block.
func containerTitle(blockName string) string {
	switch blockName {
	case "minecraft:crafting_table":
		return "Crafting"
	case "minecraft:furnace", "minecraft:lit_furnace":
		return "Furnace"
	case "minecraft:blast_furnace", "minecraft:lit_blast_furnace":
		return "Blast Furnace"
	case "minecraft:smoker", "minecraft:lit_smoker":
		return "Smoker"
	case "minecraft:anvil", "minecraft:chipped_anvil", "minecraft:damaged_anvil":
		return "Repair & Name"
	case "minecraft:enchanting_table":
		return "Enchant"
	case "minecraft:grindstone":
		return "Repair & Disenchant"
	case "minecraft:loom":
		return "Loom"
	case "minecraft:smithing_table":
		return "Upgrade Gear"
	case "minecraft:stonecutter":
		return "Stonecutter"
	case "minecraft:brewing_stand":
		return "Brewing Stand"
	case "minecraft:cartography_table":
		return "Cartography Table"
	case "minecraft:beacon":
		return "Beacon"
	case "minecraft:chest", "minecraft:trapped_chest", "minecraft:barrel":
		return "Chest"
	case "minecraft:ender_chest":
		return "Ender Chest"
	case "minecraft:hopper":
		return "Hopper"
	case "minecraft:dispenser":
		return "Dispenser"
	case "minecraft:dropper":
		return "Dropper"
	default:
		return "Container"
	}
}
