package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// MaxStringLength is the maximum number of UTF-8 bytes in a Minecraft protocol string.
	// The protocol specifies a maximum of 32767 characters × 4 bytes per char = 131068 bytes.
	MaxStringLength = 131068
)

// ReadString reads a Minecraft-encoded string: a VarInt byte-length followed by UTF-8 data.
func ReadString(r io.Reader) (string, error) {
	length, err := ReadVarInt(r)
	if err != nil {
		return "", fmt.Errorf("protocol: reading string length: %w", err)
	}
	if length < 0 {
		return "", fmt.Errorf("protocol: negative string length %d", length)
	}
	if length > MaxStringLength {
		return "", fmt.Errorf("protocol: string length %d exceeds maximum %d", length, MaxStringLength)
	}
	if length == 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("protocol: reading string body: %w", err)
	}
	return string(buf), nil
}

// WriteString encodes s as a Minecraft string (VarInt length + UTF-8 bytes).
func WriteString(w io.Writer, s string) error {
	b := []byte(s)
	if len(b) > MaxStringLength {
		return fmt.Errorf("protocol: string length %d exceeds maximum %d", len(b), MaxStringLength)
	}
	if err := WriteVarInt(w, int32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

// ReadUShort reads a big-endian unsigned 16-bit integer.
func ReadUShort(r io.Reader) (uint16, error) {
	var buf [2]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("protocol: reading ushort: %w", err)
	}
	return binary.BigEndian.Uint16(buf[:]), nil
}

// WriteUShort writes a big-endian unsigned 16-bit integer.
func WriteUShort(w io.Writer, v uint16) error {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], v)
	_, err := w.Write(buf[:])
	return err
}

// ReadLong reads a big-endian signed 64-bit integer.
func ReadLong(r io.Reader) (int64, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("protocol: reading long: %w", err)
	}
	return int64(binary.BigEndian.Uint64(buf[:])), nil
}

// WriteLong writes a big-endian signed 64-bit integer.
func WriteLong(w io.Writer, v int64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(v))
	_, err := w.Write(buf[:])
	return err
}

// ReadBool reads a single byte and returns true if it is non-zero.
func ReadBool(r io.Reader) (bool, error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return false, fmt.Errorf("protocol: reading bool: %w", err)
	}
	return buf[0] != 0, nil
}

// WriteBool writes 0x01 for true, 0x00 for false.
func WriteBool(w io.Writer, v bool) error {
	b := byte(0x00)
	if v {
		b = 0x01
	}
	_, err := w.Write([]byte{b})
	return err
}

// ReadInt reads a big-endian signed 32-bit integer.
func ReadInt(r io.Reader) (int32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("protocol: reading int: %w", err)
	}
	return int32(binary.BigEndian.Uint32(buf[:])), nil
}

// WriteInt writes a big-endian signed 32-bit integer.
func WriteInt(w io.Writer, v int32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(v))
	_, err := w.Write(buf[:])
	return err
}

// ReadByte reads a single byte.
func ReadByte(r io.Reader) (byte, error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("protocol: reading byte: %w", err)
	}
	return buf[0], nil
}

// ReadByteArray reads a VarInt-prefixed byte slice.
// The maximum accepted length is MaxPacketSize.
func ReadByteArray(r io.Reader) ([]byte, error) {
	length, err := ReadVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("protocol: reading byte array length: %w", err)
	}
	if length < 0 {
		return nil, fmt.Errorf("protocol: negative byte array length %d", length)
	}
	if length > MaxPacketSize {
		return nil, fmt.Errorf("protocol: byte array length %d exceeds maximum %d", length, MaxPacketSize)
	}
	if length == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("protocol: reading byte array body: %w", err)
	}
	return buf, nil
}

// WriteByteArray writes a VarInt-prefixed byte slice.
func WriteByteArray(w io.Writer, data []byte) error {
	if err := WriteVarInt(w, int32(len(data))); err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	_, err := w.Write(data)
	return err
}
