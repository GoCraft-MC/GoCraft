// Package protocol implements the Minecraft Java Edition wire protocol:
// packet framing, VarInt/VarLong encoding, and primitive type helpers.
package protocol

import (
	"errors"
	"io"
)

// ErrVarIntTooBig is returned when a VarInt or VarLong exceeds its maximum width.
var ErrVarIntTooBig = errors.New("protocol: VarInt too big (max 5 bytes)")

// ErrVarLongTooBig is returned when a VarLong exceeds its maximum width.
var ErrVarLongTooBig = errors.New("protocol: VarLong too big (max 10 bytes)")

// ReadVarInt reads a variable-length 32-bit integer from r.
// The Minecraft protocol encodes VarInts as groups of 7 bits, LSB first,
// with the high bit of each byte set to 1 if more bytes follow.
//
// If EOF is reached after at least one byte has been consumed (i.e. mid-varint),
// io.ErrUnexpectedEOF is returned so callers can distinguish a clean stream
// close from a truncated value.
func ReadVarInt(r io.Reader) (int32, error) {
	var (
		result int32
		shift  uint
		b      [1]byte
		read   int // bytes consumed so far
	)
	for shift < 35 {
		_, err := io.ReadFull(r, b[:])
		if err != nil {
			if err == io.EOF && read > 0 {
				// Stream ended mid-varint.
				return 0, io.ErrUnexpectedEOF
			}
			return 0, err
		}
		read++
		result |= int32(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			return result, nil
		}
		shift += 7
	}
	return 0, ErrVarIntTooBig
}

// WriteVarInt encodes v as a VarInt and writes it to w.
func WriteVarInt(w io.Writer, v int32) error {
	uv := uint32(v)
	var buf [5]byte
	n := 0
	for {
		b := byte(uv & 0x7F)
		uv >>= 7
		if uv != 0 {
			b |= 0x80
		}
		buf[n] = b
		n++
		if uv == 0 {
			break
		}
	}
	_, err := w.Write(buf[:n])
	return err
}

// VarIntSize returns the number of bytes required to encode v as a VarInt.
func VarIntSize(v int32) int {
	uv := uint32(v)
	size := 1
	for uv >= 0x80 {
		uv >>= 7
		size++
	}
	return size
}

// ReadVarLong reads a variable-length 64-bit integer from r.
func ReadVarLong(r io.Reader) (int64, error) {
	var (
		result int64
		shift  uint
		b      [1]byte
	)
	for shift < 70 {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		result |= int64(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			return result, nil
		}
		shift += 7
	}
	return 0, ErrVarLongTooBig
}

// WriteVarLong encodes v as a VarLong and writes it to w.
func WriteVarLong(w io.Writer, v int64) error {
	uv := uint64(v)
	var buf [10]byte
	n := 0
	for {
		b := byte(uv & 0x7F)
		uv >>= 7
		if uv != 0 {
			b |= 0x80
		}
		buf[n] = b
		n++
		if uv == 0 {
			break
		}
	}
	_, err := w.Write(buf[:n])
	return err
}
