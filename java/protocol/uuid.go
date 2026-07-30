package protocol

import (
	"fmt"
	"io"
)

// UUID is a Minecraft 128-bit universally unique identifier transmitted as
// two consecutive big-endian 64-bit integers (most-significant first).
type UUID [16]byte

// String returns the standard 8-4-4-4-12 hex representation.
func (u UUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// IsZero reports whether all bytes are zero (the nil UUID).
func (u UUID) IsZero() bool {
	for _, b := range u {
		if b != 0 {
			return false
		}
	}
	return true
}

// ReadUUID reads a 16-byte UUID from r (two big-endian int64s, MSB first).
func ReadUUID(r io.Reader) (UUID, error) {
	var u UUID
	if _, err := io.ReadFull(r, u[:]); err != nil {
		return UUID{}, fmt.Errorf("protocol: reading UUID: %w", err)
	}
	return u, nil
}

// WriteUUID writes the 16 bytes of u to w.
func WriteUUID(w io.Writer, u UUID) error {
	if _, err := w.Write(u[:]); err != nil {
		return fmt.Errorf("protocol: writing UUID: %w", err)
	}
	return nil
}
