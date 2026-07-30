package protocol

import (
	"bytes"
	"fmt"
	"io"
)

const (
	// MaxPacketSize is the largest accepted uncompressed packet payload (8 MiB).
	// Vanilla uses 2097152 (2 MiB) for compressed packets; we are generous for
	// status packets (registry data can be large) while still bounding memory.
	MaxPacketSize = 8 * 1024 * 1024
)

// Packet represents a decoded Minecraft protocol packet.
// ID is the numeric packet identifier for the current connection state.
// Data is the raw bytes of the packet body (everything after the packet ID).
type Packet struct {
	ID   int32
	Data []byte
}

// Reader wraps a Packet's Data slice so handlers can read fields
// sequentially without carrying a separate offset.
func (p *Packet) Reader() *bytes.Reader {
	return bytes.NewReader(p.Data)
}

// ReadPacket reads exactly one packet from r.
//
// Wire format (no compression):
//
//	VarInt  length   (number of bytes that follow: packetID bytes + data bytes)
//	VarInt  packetID
//	[]byte  data
func ReadPacket(r io.Reader) (*Packet, error) {
	length, err := ReadVarInt(r)
	if err != nil {
		return nil, fmt.Errorf("packet: reading length: %w", err)
	}
	if length < 1 {
		return nil, fmt.Errorf("packet: invalid length %d", length)
	}
	if length > MaxPacketSize {
		return nil, fmt.Errorf("packet: length %d exceeds maximum %d", length, MaxPacketSize)
	}

	// Read the full packet payload into a buffer so we can parse the VarInt ID
	// without blocking on the underlying reader mid-packet.
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("packet: reading payload: %w", err)
	}

	br := bytes.NewReader(payload)
	id, err := ReadVarInt(br)
	if err != nil {
		return nil, fmt.Errorf("packet: reading packet ID: %w", err)
	}

	data := make([]byte, br.Len())
	if _, err := io.ReadFull(br, data); err != nil {
		return nil, fmt.Errorf("packet: reading packet data: %w", err)
	}

	return &Packet{ID: id, Data: data}, nil
}

// WritePacket serialises pkt and writes it to w.
//
// Wire format (no compression):
//
//	VarInt  length   (VarInt-encoded packetID size + len(data))
//	VarInt  packetID
//	[]byte  data
func WritePacket(w io.Writer, pkt *Packet) error {
	// Encode the packet ID first so we know its byte width.
	var idBuf bytes.Buffer
	if err := WriteVarInt(&idBuf, pkt.ID); err != nil {
		return fmt.Errorf("packet: encoding packet ID: %w", err)
	}

	totalLen := int32(idBuf.Len() + len(pkt.Data))

	// Assemble the full frame in a local buffer, then write in one shot
	// to avoid partial writes on the TCP stream.
	var frame bytes.Buffer
	frame.Grow(VarIntSize(totalLen) + int(totalLen))

	if err := WriteVarInt(&frame, totalLen); err != nil {
		return fmt.Errorf("packet: encoding length: %w", err)
	}
	frame.Write(idBuf.Bytes())
	frame.Write(pkt.Data)

	_, err := w.Write(frame.Bytes())
	return err
}

// Builder is a helper for constructing a packet's Data field incrementally.
type Builder struct {
	id  int32
	buf bytes.Buffer
}

// NewBuilder returns a Builder for a packet with the given ID.
func NewBuilder(id int32) *Builder {
	return &Builder{id: id}
}

// VarInt appends a VarInt field.
func (b *Builder) VarInt(v int32) *Builder {
	if err := WriteVarInt(&b.buf, v); err != nil {
		panic(fmt.Sprintf("Builder.VarInt: %v", err))
	}
	return b
}

// String appends a Minecraft string field.
func (b *Builder) String(s string) *Builder {
	if err := WriteString(&b.buf, s); err != nil {
		panic(fmt.Sprintf("Builder.String: %v", err))
	}
	return b
}

// Long appends a big-endian int64 field.
func (b *Builder) Long(v int64) *Builder {
	if err := WriteLong(&b.buf, v); err != nil {
		panic(fmt.Sprintf("Builder.Long: %v", err))
	}
	return b
}

// Bool appends a boolean field.
func (b *Builder) Bool(v bool) *Builder {
	if err := WriteBool(&b.buf, v); err != nil {
		panic(fmt.Sprintf("Builder.Bool: %v", err))
	}
	return b
}

// Bytes appends raw bytes with no length prefix.
func (b *Builder) Bytes(data []byte) *Builder {
	b.buf.Write(data)
	return b
}

// ByteArray appends a VarInt-prefixed byte slice.
func (b *Builder) ByteArray(data []byte) *Builder {
	if err := WriteByteArray(&b.buf, data); err != nil {
		panic(fmt.Sprintf("Builder.ByteArray: %v", err))
	}
	return b
}

// UUID appends a 16-byte UUID.
func (b *Builder) UUID(u UUID) *Builder {
	if err := WriteUUID(&b.buf, u); err != nil {
		panic(fmt.Sprintf("Builder.UUID: %v", err))
	}
	return b
}

// Int appends a big-endian int32 field.
func (b *Builder) Int(v int32) *Builder {
	if err := WriteInt(&b.buf, v); err != nil {
		panic(fmt.Sprintf("Builder.Int: %v", err))
	}
	return b
}

// Byte appends a single byte field.
func (b *Builder) Byte(v byte) *Builder {
	b.buf.WriteByte(v)
	return b
}

// Float appends a big-endian IEEE 754 single-precision float.
func (b *Builder) Float(v float32) *Builder {
	if err := WriteFloat(&b.buf, v); err != nil {
		panic(fmt.Sprintf("Builder.Float: %v", err))
	}
	return b
}

// Double appends a big-endian IEEE 754 double-precision float.
func (b *Builder) Double(v float64) *Builder {
	if err := WriteDouble(&b.buf, v); err != nil {
		panic(fmt.Sprintf("Builder.Double: %v", err))
	}
	return b
}

// Build returns the assembled Packet.
func (b *Builder) Build() *Packet {
	return &Packet{ID: b.id, Data: b.buf.Bytes()}
}
