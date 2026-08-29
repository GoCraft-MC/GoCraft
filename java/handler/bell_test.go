package handler

import (
	"testing"

	"GoCraft/core/spatial"
	"GoCraft/java/protocol"
	javaworld "GoCraft/java/world"
)

func TestJavaBellBlockEventUsesTransientDirection(t *testing.T) {
	position := spatial.BlockPos{X: 2, Y: 64, Z: -3}
	packets := buildBellRingPackets(position, "east")
	if len(packets) != 2 || packets[0].ID != packetIDBlockAction {
		t.Fatalf("packets=%v", packets)
	}
	r := packets[0].Reader()
	packed, err := protocol.ReadLong(r)
	if err != nil || packed != position.Encode() {
		t.Fatalf("position=%d err=%v, want %d", packed, err, position.Encode())
	}
	action, _ := protocol.ReadByte(r)
	direction, _ := protocol.ReadByte(r)
	blockID, _ := protocol.ReadVarInt(r)
	wantBellID, ok := javaworld.BlockTypeID("minecraft:bell")
	if !ok || action != 1 || direction != 5 || blockID != wantBellID {
		t.Fatalf("action=%d direction=%d block=%d, want 1/5/%d", action, direction, blockID, wantBellID)
	}
}
