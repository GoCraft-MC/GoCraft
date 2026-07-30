package handler

// Built-in command implementations for Milestone 12.
//
// Each command receives a CommandContext and returns an error whose text is
// shown to the issuing player.  The Dispatcher catches all returns.
//
// Commands implemented here:
//   /help                              — list available commands
//   /list                              — list online players
//   /gamemode <mode>                   — change the issuing player's game mode
//   /tp <x> <y> <z>                   — teleport to coordinates
//   /tp <player>                       — teleport to another player
//   /give <item> [count]               — add an item to the hotbar / inventory
//   /kick <player> [reason]            — disconnect a player
//
// Packet IDs marked "estimate" should be verified against a 1.21.4 capture.

import (
	"fmt"
	"strconv"
	"strings"

	"GoCraft/core/player"
	"GoCraft/java/protocol"
)

// packetIDDisconnectPlay is the S→C Disconnect packet in the Play state.
// Estimate for 1.21.4 (protocol 769); verify against packet dump.
const packetIDDisconnectPlay = 0x1D

// RegisterBuiltins registers all built-in GoCraft commands with d.
func RegisterBuiltins(d *Dispatcher) {
	d.Register("help", cmdHelp)
	d.Register("list", cmdList)
	d.Register("gamemode", cmdGameMode)
	d.Register("gm", cmdGameMode) // short alias
	d.Register("tp", cmdTp)
	d.Register("give", cmdGive)
	d.Register("kick", cmdKick)
}

// ── /help ─────────────────────────────────────────────────────────────────────

func cmdHelp(ctx CommandContext) error {
	_ = sendSystemMessage(ctx.Conn,
		"Commands: /gamemode /tp /give /kick /list /help")
	return nil
}

// ── /list ─────────────────────────────────────────────────────────────────────

func cmdList(ctx CommandContext) error {
	sessions := ctx.Manager.SnapshotAll()
	names := make([]string, 0, len(sessions))
	for _, s := range sessions {
		names = append(names, s.Player.Username)
	}
	_ = sendSystemMessage(ctx.Conn,
		fmt.Sprintf("Online (%d): %s", len(names), strings.Join(names, ", ")))
	return nil
}

// ── /gamemode ─────────────────────────────────────────────────────────────────

func cmdGameMode(ctx CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: /gamemode <survival|creative|adventure|spectator>")
	}
	var mode player.GameMode
	switch strings.ToLower(ctx.Args[0]) {
	case "survival", "s", "0":
		mode = player.GameModeSurvival
	case "creative", "c", "1":
		mode = player.GameModeCreative
	case "adventure", "a", "2":
		mode = player.GameModeAdventure
	case "spectator", "sp", "3":
		mode = player.GameModeSpectator
	default:
		return fmt.Errorf("unknown game mode: %q", ctx.Args[0])
	}
	ctx.Player.GameMode = mode

	// Notify the client of its new game mode via a Game Event.
	// Reason 3 = change_game_mode; value = mode as float32.
	if err := sendGameEvent(ctx.Conn, 3, float32(mode)); err != nil {
		return fmt.Errorf("sending game event: %w", err)
	}

	// Resync flight / speed / instant-break flags for the new mode.
	if err := sendPlayerAbilities(ctx.Conn, ctx.Player); err != nil {
		return fmt.Errorf("sending abilities: %w", err)
	}

	// Update the tab-list game mode for all connected players.
	updatePkt := buildGameModeUpdate(ctx.Player)
	for _, s := range ctx.Manager.SnapshotAll() {
		_ = s.Conn.WritePacket(updatePkt)
	}

	modeName := [4]string{"survival", "creative", "adventure", "spectator"}[mode]
	_ = sendSystemMessage(ctx.Conn, "Game mode changed to "+modeName)
	return nil
}

// buildGameModeUpdate builds a Player Info Update (action 0x04 = UPDATE_GAME_MODE)
// packet to update p's game mode entry in every client's tab list.
//
// Wire layout (1.21.4):
//
//	Byte    actions        = 0x04 (UPDATE_GAME_MODE)
//	VarInt  player_count   = 1
//	UUID    player_uuid
//	VarInt  game_mode
func buildGameModeUpdate(p *player.Player) *protocol.Packet {
	return protocol.NewBuilder(packetIDPlayerInfoUpdate).
		Byte(0x04). // UPDATE_GAME_MODE action mask
		VarInt(1).
		UUID(protocol.UUID(p.UUID)).
		VarInt(int32(p.GameMode)).
		Build()
}

// ── /tp ───────────────────────────────────────────────────────────────────────

func cmdTp(ctx CommandContext) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("usage: /tp <x> <y> <z>  or  /tp <player>")
	}

	if len(ctx.Args) >= 3 {
		// Coordinate teleport.
		x, err := strconv.ParseFloat(ctx.Args[0], 64)
		if err != nil {
			return fmt.Errorf("invalid x: %q", ctx.Args[0])
		}
		y, err := strconv.ParseFloat(ctx.Args[1], 64)
		if err != nil {
			return fmt.Errorf("invalid y: %q", ctx.Args[1])
		}
		z, err := strconv.ParseFloat(ctx.Args[2], 64)
		if err != nil {
			return fmt.Errorf("invalid z: %q", ctx.Args[2])
		}
		if err := ctx.TeleportTo(x, y, z); err != nil {
			return fmt.Errorf("teleporting: %w", err)
		}
		_ = sendSystemMessage(ctx.Conn,
			fmt.Sprintf("Teleported to %.2f %.2f %.2f", x, y, z))
		return nil
	}

	// Player-name teleport.
	targetName := ctx.Args[0]
	for _, s := range ctx.Manager.SnapshotAll() {
		if strings.EqualFold(s.Player.Username, targetName) {
			pos := s.Player.Position
			if err := ctx.TeleportTo(pos.X, pos.Y, pos.Z); err != nil {
				return fmt.Errorf("teleporting: %w", err)
			}
			_ = sendSystemMessage(ctx.Conn,
				fmt.Sprintf("Teleported to %s", s.Player.Username))
			return nil
		}
	}
	return fmt.Errorf("player not found: %s", targetName)
}

// ── /give ─────────────────────────────────────────────────────────────────────

func cmdGive(ctx CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: /give <item> [count]")
	}

	// Normalize item name: accept "stone" or "minecraft:stone".
	itemName := ctx.Args[0]
	if !strings.Contains(itemName, ":") {
		itemName = "minecraft:" + itemName
	}

	count := 1
	if len(ctx.Args) >= 2 {
		n, err := strconv.Atoi(ctx.Args[1])
		if err != nil || n < 1 {
			return fmt.Errorf("count must be a positive integer, got %q", ctx.Args[1])
		}
		count = n
	}
	if count > 64 {
		count = 64
	}

	// Find the first empty slot: hotbar first, then main inventory.
	slot := firstEmptySlot(ctx.Player)
	if slot < 0 {
		return fmt.Errorf("inventory is full")
	}

	ctx.Player.Inventory[slot] = player.ItemStack{ItemID: itemName, Count: count}

	// Sync the full inventory so the client reflects the new item.
	if err := sendSetContainerContent(ctx.Conn, ctx.Player, 1); err != nil {
		return fmt.Errorf("syncing inventory: %w", err)
	}
	_ = sendSystemMessage(ctx.Conn,
		fmt.Sprintf("Given %dx %s", count, itemName))
	return nil
}

// firstEmptySlot returns the index of the first empty slot in the player's
// hotbar (preferred) or main inventory, or -1 if the inventory is full.
func firstEmptySlot(p *player.Player) int {
	// Hotbar: slots HotbarStart … HotbarStart+8
	for i := player.HotbarStart; i < player.HotbarStart+9; i++ {
		if p.Inventory[i].IsEmpty() {
			return i
		}
	}
	// Main inventory: slots 9 … HotbarStart-1
	for i := 9; i < player.HotbarStart; i++ {
		if p.Inventory[i].IsEmpty() {
			return i
		}
	}
	return -1
}

// ── /kick ─────────────────────────────────────────────────────────────────────

func cmdKick(ctx CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: /kick <player> [reason]")
	}
	targetName := ctx.Args[0]
	reason := "Kicked by an operator"
	if len(ctx.Args) >= 2 {
		reason = strings.Join(ctx.Args[1:], " ")
	}

	for _, s := range ctx.Manager.SnapshotAll() {
		if strings.EqualFold(s.Player.Username, targetName) {
			// Send a Disconnect packet so the client shows the reason, then
			// close the connection.  The play loop will clean up the session
			// via the deferred onPlayerLeave / mgr.Remove calls.
			_ = s.Conn.WritePacket(buildDisconnectPlay(reason))
			_ = s.Conn.Close()
			_ = sendSystemMessage(ctx.Conn,
				fmt.Sprintf("Kicked %s: %s", s.Player.Username, reason))
			return nil
		}
	}
	return fmt.Errorf("player not found: %s", targetName)
}

// buildDisconnectPlay constructs a Disconnect (Play) S→C packet.
//
// Wire layout (1.21.4):
//
//	Text Component (NBT)  reason
//
// The reason is encoded as a Network NBT text component (same format used by
// System Chat Message since 1.20.3).
func buildDisconnectPlay(reason string) *protocol.Packet {
	return protocol.NewBuilder(packetIDDisconnectPlay).
		Bytes(nbtTextComponent(reason)).
		Build()
}
