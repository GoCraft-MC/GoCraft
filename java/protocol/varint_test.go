package protocol

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// varintCase is a single round-trip test case.
type varintCase struct {
	value     int32
	wantBytes []byte // expected encoded representation
}

var varintCases = []varintCase{
	{0, []byte{0x00}},
	{1, []byte{0x01}},
	{127, []byte{0x7F}},
	{128, []byte{0x80, 0x01}},
	{255, []byte{0xFF, 0x01}},
	{2147483647, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x07}},  // max int32
	{-1, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}},          // two's complement
	{-2147483648, []byte{0x80, 0x80, 0x80, 0x80, 0x08}}, // min int32
}

func TestWriteVarInt(t *testing.T) {
	for _, tc := range varintCases {
		var buf bytes.Buffer
		if err := WriteVarInt(&buf, tc.value); err != nil {
			t.Errorf("WriteVarInt(%d): unexpected error: %v", tc.value, err)
			continue
		}
		if !bytes.Equal(buf.Bytes(), tc.wantBytes) {
			t.Errorf("WriteVarInt(%d): got %x, want %x", tc.value, buf.Bytes(), tc.wantBytes)
		}
	}
}

func TestReadVarInt(t *testing.T) {
	for _, tc := range varintCases {
		r := bytes.NewReader(tc.wantBytes)
		got, err := ReadVarInt(r)
		if err != nil {
			t.Errorf("ReadVarInt(%x): unexpected error: %v", tc.wantBytes, err)
			continue
		}
		if got != tc.value {
			t.Errorf("ReadVarInt(%x): got %d, want %d", tc.wantBytes, got, tc.value)
		}
	}
}

func TestVarIntRoundTrip(t *testing.T) {
	values := []int32{0, 1, -1, 127, 128, 256, 1000, 2147483647, -2147483648, 300}
	for _, v := range values {
		var buf bytes.Buffer
		if err := WriteVarInt(&buf, v); err != nil {
			t.Fatalf("WriteVarInt(%d): %v", v, err)
		}
		got, err := ReadVarInt(&buf)
		if err != nil {
			t.Fatalf("ReadVarInt after WriteVarInt(%d): %v", v, err)
		}
		if got != v {
			t.Errorf("round-trip %d: got %d", v, got)
		}
	}
}

func TestReadVarInt_TooBig(t *testing.T) {
	// 6 bytes all with continuation bit set — exceeds max 5 bytes for VarInt.
	bad := bytes.NewReader([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80})
	_, err := ReadVarInt(bad)
	if !errors.Is(err, ErrVarIntTooBig) {
		t.Errorf("expected ErrVarIntTooBig, got %v", err)
	}
}

func TestReadVarInt_UnexpectedEOF(t *testing.T) {
	// Byte with continuation bit but no following byte.
	bad := bytes.NewReader([]byte{0x80})
	_, err := ReadVarInt(bad)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("expected io.ErrUnexpectedEOF, got %v", err)
	}
}

func TestVarIntSize(t *testing.T) {
	cases := []struct {
		v    int32
		size int
	}{
		{0, 1},
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
		{2097151, 3},
		{2097152, 4},
		{268435455, 4},
		{268435456, 5},
		{2147483647, 5},
		{-1, 5}, // all bits set -> 5 bytes
	}
	for _, tc := range cases {
		if got := VarIntSize(tc.v); got != tc.size {
			t.Errorf("VarIntSize(%d): got %d, want %d", tc.v, got, tc.size)
		}
	}
}

// --- VarLong tests ---

type varlongCase struct {
	value     int64
	wantBytes []byte
}

var varlongCases = []varlongCase{
	{0, []byte{0x00}},
	{1, []byte{0x01}},
	{127, []byte{0x7F}},
	{128, []byte{0x80, 0x01}},
	{9223372036854775807, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x7F}}, // max int64
	{-1, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01}},
}

func TestVarLongRoundTrip(t *testing.T) {
	for _, tc := range varlongCases {
		var buf bytes.Buffer
		if err := WriteVarLong(&buf, tc.value); err != nil {
			t.Fatalf("WriteVarLong(%d): %v", tc.value, err)
		}
		if !bytes.Equal(buf.Bytes(), tc.wantBytes) {
			t.Errorf("WriteVarLong(%d): got %x, want %x", tc.value, buf.Bytes(), tc.wantBytes)
		}
		got, err := ReadVarLong(bytes.NewReader(buf.Bytes()))
		if err != nil {
			t.Fatalf("ReadVarLong after WriteVarLong(%d): %v", tc.value, err)
		}
		if got != tc.value {
			t.Errorf("VarLong round-trip %d: got %d", tc.value, got)
		}
	}
}
