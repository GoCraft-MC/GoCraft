package world

import (
	"testing"

	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
)

func TestBuildBlockLightUpdateEncodesJavaLightPayload(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(8, 64, 8, coreworld.Block{Namespace: "minecraft", Name: "torch"})
	packet := BuildBlockLightUpdate(world, world.Chunk(0, 0))
	if packet == nil || packet.ID != packetIDUpdateLight {
		t.Fatalf("packet = %+v, want Java update_light ID %d", packet, packetIDUpdateLight)
	}

	reader := packet.Reader()
	for _, field := range []string{"chunk x", "chunk z"} {
		if _, err := protocol.ReadVarInt(reader); err != nil {
			t.Fatalf("read %s: %v", field, err)
		}
	}
	if count := readLightBitSet(t, reader); count != 0 {
		t.Fatalf("sky light mask longs = %d, want 0", count)
	}
	if count := readLightBitSet(t, reader); count != 1 {
		t.Fatalf("block light mask longs = %d, want 1", count)
	}
	if count := readLightBitSet(t, reader); count != 0 {
		t.Fatalf("empty sky light mask longs = %d, want 0", count)
	}
	if count := readLightBitSet(t, reader); count != 1 {
		t.Fatalf("empty block light mask longs = %d, want 1", count)
	}
	if count, err := protocol.ReadVarInt(reader); err != nil || count != 0 {
		t.Fatalf("sky array count = %d, err=%v, want 0", count, err)
	}
	blockArrays, err := protocol.ReadVarInt(reader)
	if err != nil || blockArrays < 1 {
		t.Fatalf("block array count = %d, err=%v, want at least 1", blockArrays, err)
	}
	for index := int32(0); index < blockArrays; index++ {
		light, readErr := protocol.ReadByteArray(reader)
		if readErr != nil || len(light) != 2048 {
			t.Fatalf("block array %d length = %d, err=%v, want 2048", index, len(light), readErr)
		}
	}
	if reader.Len() != 0 {
		t.Fatalf("update_light has %d trailing bytes", reader.Len())
	}
}

func readLightBitSet(t *testing.T, reader interface {
	Read([]byte) (int, error)
}) int32 {
	t.Helper()
	count, err := protocol.ReadVarInt(reader)
	if err != nil {
		t.Fatalf("read light bit-set length: %v", err)
	}
	for index := int32(0); index < count; index++ {
		if _, err := protocol.ReadLong(reader); err != nil {
			t.Fatalf("read light bit-set long %d: %v", index, err)
		}
	}
	return count
}
