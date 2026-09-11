package handler

import (
	"fmt"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/java/protocol"
)

func handleLecternButtonClick(pkt *protocol.Packet, p *player.Player, bus *intent.Bus) error {
	r := pkt.Reader()
	windowID, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("lectern button: reading window ID: %w", err)
	}
	buttonID, err := protocol.ReadVarInt(r)
	if err != nil {
		return fmt.Errorf("lectern button: reading button ID: %w", err)
	}
	if r.Len() != 0 || p == nil || bus == nil || windowID != chestContainerID ||
		p.OpenContainerID != windowID || p.OpenContainerKind != "minecraft:lectern" {
		return nil
	}
	page, relative := 0, false
	switch {
	case buttonID == 1:
		page, relative = -1, true
	case buttonID == 2:
		page, relative = 1, true
	case buttonID >= 100:
		page = int(buttonID - 100)
	default:
		return nil
	}
	bus.PostLecternPage(intent.LecternPageIntent{
		PlayerUUID: p.UUID, Dimension: p.Dimension, Position: p.OpenContainerPos,
		Page: page, Relative: relative,
	})
	return nil
}
