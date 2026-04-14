package protocol

import (
	"bytes"
	"testing"
)

func TestWriteReadPacket_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		pkt  *Packet
	}{
		{
			name: "empty data",
			pkt:  &Packet{ID: 0x00, Data: []byte{}},
		},
		{
			name: "status request",
			pkt:  &Packet{ID: 0x00, Data: []byte{}},
		},
		{
			name: "ping payload",
			pkt:  &Packet{ID: 0x01, Data: []byte{0x00, 0x00, 0x00, 0x00, 0xDE, 0xAD, 0xBE, 0xEF}},
		},
		{
			name: "arbitrary data",
			pkt:  &Packet{ID: 0x10, Data: []byte{0x01, 0x02, 0x03, 0xFF}},
		},
		{
			name: "large packet ID",
			pkt:  &Packet{ID: 0x7FFF, Data: []byte{0xAB, 0xCD}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			if err := WritePacket(&buf, tc.pkt); err != nil {
				t.Fatalf("WritePacket: %v", err)
			}

			got, err := ReadPacket(&buf)
			if err != nil {
				t.Fatalf("ReadPacket: %v", err)
			}

			if got.ID != tc.pkt.ID {
				t.Errorf("ID: got %d, want %d", got.ID, tc.pkt.ID)
			}
			if !bytes.Equal(got.Data, tc.pkt.Data) {
				t.Errorf("Data: got %x, want %x", got.Data, tc.pkt.Data)
			}
		})
	}
}

func TestReadPacket_InvalidLength(t *testing.T) {
	// Length = 0 is invalid (at least 1 byte needed for packet ID).
	buf := bytes.NewReader([]byte{0x00})
	_, err := ReadPacket(buf)
	if err == nil {
		t.Error("expected error for zero-length packet, got nil")
	}
}

func TestReadPacket_ExceedsMaxSize(t *testing.T) {
	// Encode a length varint that exceeds MaxPacketSize without actual data.
	var buf bytes.Buffer
	// MaxPacketSize + 1 encoded as VarInt
	oversized := int32(MaxPacketSize + 1)
	if err := WriteVarInt(&buf, oversized); err != nil {
		t.Fatalf("WriteVarInt: %v", err)
	}
	_, err := ReadPacket(&buf)
	if err == nil {
		t.Error("expected error for oversized packet, got nil")
	}
}

func TestBuilder(t *testing.T) {
	b := NewBuilder(0x00).
		String("hello").
		VarInt(42).
		Long(12345678).
		Bool(true)

	pkt := b.Build()
	if pkt.ID != 0x00 {
		t.Errorf("Builder ID: got %d, want 0", pkt.ID)
	}

	r := pkt.Reader()

	// Read back String
	s, err := ReadString(r)
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if s != "hello" {
		t.Errorf("String: got %q, want \"hello\"", s)
	}

	// Read back VarInt
	vi, err := ReadVarInt(r)
	if err != nil {
		t.Fatalf("ReadVarInt: %v", err)
	}
	if vi != 42 {
		t.Errorf("VarInt: got %d, want 42", vi)
	}

	// Read back Long
	l, err := ReadLong(r)
	if err != nil {
		t.Fatalf("ReadLong: %v", err)
	}
	if l != 12345678 {
		t.Errorf("Long: got %d, want 12345678", l)
	}

	// Read back Bool
	bl, err := ReadBool(r)
	if err != nil {
		t.Fatalf("ReadBool: %v", err)
	}
	if !bl {
		t.Error("Bool: got false, want true")
	}
}

func TestPacket_FrameIntegrity(t *testing.T) {
	// Verify that two packets written sequentially can be read back independently.
	var buf bytes.Buffer

	p1 := &Packet{ID: 0x01, Data: []byte("first")}
	p2 := &Packet{ID: 0x02, Data: []byte("second")}

	if err := WritePacket(&buf, p1); err != nil {
		t.Fatalf("WritePacket p1: %v", err)
	}
	if err := WritePacket(&buf, p2); err != nil {
		t.Fatalf("WritePacket p2: %v", err)
	}

	got1, err := ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket p1: %v", err)
	}
	if got1.ID != p1.ID || !bytes.Equal(got1.Data, p1.Data) {
		t.Errorf("p1 mismatch: got {%d, %x}, want {%d, %x}", got1.ID, got1.Data, p1.ID, p1.Data)
	}

	got2, err := ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket p2: %v", err)
	}
	if got2.ID != p2.ID || !bytes.Equal(got2.Data, p2.Data) {
		t.Errorf("p2 mismatch: got {%d, %x}, want {%d, %x}", got2.ID, got2.Data, p2.ID, p2.Data)
	}
}
