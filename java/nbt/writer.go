package nbt

import (
	"bytes"
	"encoding/binary"
)

type Writer struct {
	Buf *bytes.Buffer
}

func NewWriterWithBuffer() Writer {
	return Writer{
		Buf: &bytes.Buffer{},
	}
}

func (w Writer) Bytes() []byte {
	return w.Buf.Bytes()
}

func (w Writer) writeBytes(data []byte) {
	w.Buf.Write(data)
}

func (w Writer) writeUInt16(value int) {
	var data [2]byte
	binary.BigEndian.PutUint16(data[:], uint16(value))
	w.writeBytes(data[:])
}

func (w Writer) WriteByte(data byte) error {
	return w.Buf.WriteByte(data)
}

func (w Writer) WriteString(value string) {
	nameBytes := []byte(value)
	w.writeUInt16(len(value))
	w.writeBytes(nameBytes)
}

func (w Writer) WriteStringEntry(name string, value string) {
	w.Buf.WriteByte(0x08) // TAG_String
	w.WriteString(name)
	w.WriteString(value)
}
