package nbt

// nbtTextComponent encodes a plain text string as a minimal Network NBT text
// component, as required by System Chat Message and other packets in 1.20.3+.
//
// Encoding: a root TAG_Compound (no name — network NBT format) containing a
// single TAG_String field named "text", followed by TAG_End.
//
//	0x0A              TAG_Compound root (no name)
//	0x08 "text" …    TAG_String: name="text", value=text
//	0x00              TAG_End
type TextComponent struct {
	Text string
}

func (c TextComponent) Bytes() []byte {
	w := NewWriterWithBuffer()
	w.WriteByte(0x0A) // TAG_Compound root
	w.WriteStringEntry("text", c.Text)
	w.WriteByte(0x00) // TAG_End
	return w.Bytes()
}
