package nbt

const (
	OpenUrlTextLinkAction = "open_url"
)

type TextLinkComponentAction struct {
	Action string
	Value  string
}

func OpenUrlAction(value string) TextLinkComponentAction {
	return TextLinkComponentAction{
		Action: OpenUrlTextLinkAction,
		Value:  value,
	}
}

const (
	TextLinkUnderlinedStyle = "underlined"

	TextColorAqua = "aqua"
)

type TextLinkComponent struct {
	Prefix string // plain non-clickable label shown before the link
	Text   string // visible clickable text (the URL)
	Color  string
	Style  string
	Action TextLinkComponentAction
}

// DefaultTextLinkComponent builds a component that shows prefix as plain text
// followed by link as a visible, clickable, aqua-underlined URL.
func DefaultTextLinkComponent(prefix string, link string) TextLinkComponent {
	return TextLinkComponent{
		Prefix: prefix + " ",
		Text:   link,
		Color:  TextColorAqua,
		Style:  TextLinkUnderlinedStyle,
		Action: OpenUrlAction(link),
	}
}

func (c TextLinkComponent) Bytes() []byte {
	w := NewWriterWithBuffer()
	w.WriteByte(0x0A) // root TAG_Compound (unnamed)
	w.WriteStringEntry("text", c.Prefix)
	// extra: list of one compound containing the clickable URL
	w.WriteByte(0x09)          // TAG_List
	w.WriteString("extra")     // name
	w.WriteByte(0x0A)          // element type = TAG_Compound
	w.WriteInt32(1)            // 1 element
	// element 0 — clickable link (compound body, no type/name header in list)
	w.WriteStringEntry("text", c.Text)
	w.WriteStringEntry("color", c.Color)
	w.WriteByte(0x01)      // TAG_Byte
	w.WriteString(c.Style) // "underlined"
	w.WriteByte(1)         // true
	w.WriteByte(0x0A)      // TAG_Compound
	w.WriteString("clickEvent")
	w.WriteStringEntry("action", c.Action.Action)
	w.WriteStringEntry("value", c.Action.Value)
	w.WriteByte(0x00) // end clickEvent
	w.WriteByte(0x00) // end element 0
	w.WriteByte(0x00) // end root
	return w.Bytes()
}
