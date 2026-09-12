package nbt

type LoreTextComponent struct {
	Text  string
	Color string
}

// nbtLoreTextComponent encodes an explicitly styled lore line. Item lore is
// dark-purple and italic by default, so both fields must be present to produce
// the compact vanilla-like tooltip used by GoCraft.
func (c LoreTextComponent) Bytes() []byte {
	w := NewWriterWithBuffer()
	w.WriteByte(0x0A) // TAG_Compound root
	w.WriteStringEntry("text", c.Text)
	w.WriteStringEntry("color", c.Color)
	w.WriteByte(0x01) // TAG_Byte
	w.WriteString("italic")
	w.WriteByte(0) // false
	w.WriteByte(0x00)
	return w.Bytes()
}
