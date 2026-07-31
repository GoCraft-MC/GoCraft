package handler

// trade.go implements villager trading for Java Edition 1.21.4.
//
// When a player right-clicks (INTERACT or INTERACT_AT) a villager entity, the
// server:
//  1. Sends Open Screen (0x36) to open the Merchant UI (container type 19).
//  2. Sends Merchant Offers (0x2F) with the trade list.
//
// The trade list is static per biome — no persistence is needed for a basic
// implementation.  Each trade is a cheap input-item → output-item mapping; the
// client renders prices, use counters, and restock indicators automatically.

import (
	"fmt"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	javaworld "GoCraft/java/world"
)

// merchantContainerType is the protocol ID for minecraft:merchant in the
// minecraft:menu registry (confirmed from registries.json).
const merchantContainerType = int32(19)

// villagerWindowID is the fixed merchant window ID used for all villager UIs.
// A real implementation should use a per-player sequential counter; the simple
// fixed value works because only one GUI can be open at a time per player.
const villagerWindowID = int32(1)

// tradeOffer defines a single merchant trade: items the player pays (up to two
// inputs) and the item they receive.
type tradeOffer struct {
	input1     tradeItem
	input2     tradeItem // zero-value means no second input
	output     tradeItem
	maxUses    int32
	xpPerTrade int32
}

type tradeItem struct {
	itemName string // canonical resource location, e.g. "minecraft:wheat"
	count    int32
}

// defaultVillagerTrades returns the static trade list shown for all villagers.
// For a biome-aware implementation, pass the villager biome to vary the trades.
var defaultVillagerTrades = []tradeOffer{
	// Farmer: sell wheat to the villager for emeralds
	{
		input1:     tradeItem{"minecraft:wheat", 20},
		output:     tradeItem{"minecraft:emerald", 1},
		maxUses:    12,
		xpPerTrade: 5,
	},
	// Farmer: buy bread with an emerald
	{
		input1:     tradeItem{"minecraft:emerald", 1},
		output:     tradeItem{"minecraft:bread", 6},
		maxUses:    12,
		xpPerTrade: 5,
	},
	// Librarian: sell paper for emeralds
	{
		input1:     tradeItem{"minecraft:paper", 24},
		output:     tradeItem{"minecraft:emerald", 1},
		maxUses:    12,
		xpPerTrade: 3,
	},
	// General: buy an apple with an emerald
	{
		input1:     tradeItem{"minecraft:emerald", 1},
		output:     tradeItem{"minecraft:apple", 4},
		maxUses:    12,
		xpPerTrade: 3,
	},
	// Farmer: sell carrots for emeralds
	{
		input1:     tradeItem{"minecraft:carrot", 22},
		output:     tradeItem{"minecraft:emerald", 1},
		maxUses:    12,
		xpPerTrade: 3,
	},
}

// handleInteractPacket parses a C→S Interact (0x19) packet.
//
// Wire layout (1.21.4):
//
//	VarInt  entity_id
//	VarInt  type  (0=INTERACT, 1=ATTACK, 2=INTERACT_AT)
//	Float×3 target_x/y/z  (only if type == 2)
//	VarInt  hand  (0=MAIN, 1=OFF; only if type == 0 or 2)
//	Bool    sneaking
//
// If the targeted entity is a villager and the interaction is INTERACT with the
// main hand, the trading UI is opened.
func handleInteractPacket(pkt *protocol.Packet, w *coreworld.World, conn *network.ClientConn) error {
	r := pkt.Reader()

	entityID, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("interact: reading entity id: %w", err)
	}
	interactType, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("interact: reading type: %w", err)
	}

	// INTERACT_AT (2) carries three target floats before the hand field.
	if interactType == 2 {
		for i := 0; i < 3; i++ {
			if _, err := protocol.ReadFloat(r); err != nil {
				return fmt.Errorf("interact: reading target coord: %w", err)
			}
		}
	}

	// ATTACK (1) has no hand field — skip hand parsing.
	if interactType == 0 || interactType == 2 {
		hand, err := protocol.ReadVarInt(r)
		if err != nil {
			return fmt.Errorf("interact: reading hand: %w", err)
		}
		if hand != 0 {
			return nil // off-hand interact — ignore
		}
	}

	if interactType == 1 {
		return nil // attack — not handled here
	}

	// Look up entity; only villagers open a trade screen.
	entity, ok := w.Entities.Get(entityID)
	if !ok || entity.Type != corentity.TypeVillager {
		return nil
	}

	if err := sendOpenScreen(conn, villagerWindowID, merchantContainerType, "Villager"); err != nil {
		return fmt.Errorf("interact: opening screen: %w", err)
	}
	if err := sendMerchantOffers(conn, villagerWindowID, defaultVillagerTrades); err != nil {
		return fmt.Errorf("interact: sending offers: %w", err)
	}
	return nil
}

// sendOpenScreen sends the Open Screen packet (0x36 S→C).
//
// Wire layout (1.21.4):
//
//	VarInt         window_id
//	VarInt         window_type  (container type index from minecraft:menu registry)
//	Text Component title        (Network NBT format, same as System Chat)
func sendOpenScreen(conn *network.ClientConn, windowID, windowType int32, title string) error {
	pkt := protocol.NewBuilder(packetIDOpenScreen).
		VarInt(windowID).
		VarInt(windowType).
		Bytes(nbtTextComponent(title)).
		Build()
	return conn.WritePacket(pkt)
}

// sendMerchantOffers sends the Merchant Offers packet (0x2F S→C).
//
// Wire layout (1.21.4):
//
//	VarInt  window_id
//	VarInt  size  (number of trades)
//	For each trade:
//	  Slot    input_item_1
//	  Slot    output_item
//	  Bool    has_second_input
//	  [Slot   input_item_2  — only when has_second_input]
//	  Bool    out_of_stock
//	  Int     number_of_trades_uses
//	  Int     max_uses
//	  Int     xp
//	  Int     special_price
//	  Float   price_multiplier
//	  Int     demand
//	VarInt  villager_level   (1–5)
//	VarInt  villager_xp
//	Bool    is_regular_villager
//	Bool    can_restock
func sendMerchantOffers(conn *network.ClientConn, windowID int32, trades []tradeOffer) error {
	b := protocol.NewBuilder(packetIDMerchantOffers).
		VarInt(windowID).
		VarInt(int32(len(trades)))

	for _, t := range trades {
		encodeTradingSlot(b, t.input1)
		encodeTradingSlot(b, t.output)

		hasSecond := t.input2.itemName != ""
		b.Bool(hasSecond)
		if hasSecond {
			encodeTradingSlot(b, t.input2)
		}

		b.Bool(false). // out_of_stock (always available)
				Int(0).            // number_of_trades_uses
				Int(t.maxUses).    // max_uses
				Int(t.xpPerTrade). // xp earned per trade
				Int(0).            // special_price (demand discount)
				Float(0.05).       // price_multiplier
				Int(0)             // demand
	}

	b.VarInt(1). // villager_level: Novice
			VarInt(0).  // villager_xp: 0
			Bool(true). // is_regular_villager
			Bool(true)  // can_restock

	return conn.WritePacket(b.Build())
}

// encodeTradingSlot encodes a tradeItem as a 1.21.4 slot into b.
// Empty item name → empty slot (VarInt 0).
func encodeTradingSlot(b *protocol.Builder, item tradeItem) {
	if item.itemName == "" || item.count <= 0 {
		b.VarInt(0) // empty slot
		return
	}
	id := javaworld.ItemID(item.itemName)
	if id < 0 {
		b.VarInt(0) // unknown item — send empty rather than corrupt ID
		return
	}
	b.VarInt(item.count).
		VarInt(id).
		VarInt(0). // components_to_add
		VarInt(0)  // components_to_remove
}
