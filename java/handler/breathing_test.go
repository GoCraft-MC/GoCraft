package handler

import (
	"testing"

	"GoCraft/core/player"
	"GoCraft/java/protocol"
)

func TestPlayerAirSupplyUsesEntityMetadataIndexOne(t *testing.T) {
	p := player.New([16]byte{}, "diver", player.ClientEditionJava)
	p.EntityID = 42
	p.AirSupply = 123
	r := buildPlayerAirSupply(p).Reader()
	entityID, _ := protocol.ReadVarInt(r)
	index, _ := protocol.ReadByte(r)
	serializer, _ := protocol.ReadVarInt(r)
	air, _ := protocol.ReadVarInt(r)
	terminator, _ := protocol.ReadByte(r)
	if entityID != 42 || index != 1 || serializer != 1 || air != 123 || terminator != 0xff {
		t.Fatalf("air metadata = %d/%d/%d/%d/%x", entityID, index, serializer, air, terminator)
	}
}
