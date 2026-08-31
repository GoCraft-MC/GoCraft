package pluginapi

import (
	"testing"

	abi "GoCraft/abi/v1"
)

func TestWireVocabularyDecoding(t *testing.T) {
	uuid := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	player, err := playerFrom(abi.List(abi.Bytes(uuid), abi.String("Elias"), abi.String("bedrock")))
	if err != nil || player.Username != "Elias" || player.Edition != "bedrock" || player.UUID[15] != 16 {
		t.Fatalf("playerFrom() = %#v, %v", player, err)
	}
	position, err := positionFrom(abi.List(abi.Int64(12), abi.Int64(64), abi.Int64(-8)))
	if err != nil || position != (BlockPos{X: 12, Y: 64, Z: -8}) {
		t.Fatalf("positionFrom() = %#v, %v", position, err)
	}
	block, err := blockFrom(abi.List(abi.String("minecraft:stone"),
		abi.List(abi.List(abi.String("variant"), abi.String("smooth")))))
	if err != nil || block.ID != "minecraft:stone" || block.Properties["variant"] != "smooth" {
		t.Fatalf("blockFrom() = %#v, %v", block, err)
	}
}

func TestWireVocabularyRejectsMalformedValues(t *testing.T) {
	if _, err := playerFrom(abi.List()); err == nil {
		t.Fatal("malformed player was accepted")
	}
	if _, err := permissionsFrom(abi.String("admin")); err == nil {
		t.Fatal("malformed permissions were accepted")
	}
}
