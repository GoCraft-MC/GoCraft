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
	Text   string
	Color  string
	Style  string
	Action TextLinkComponentAction
}

func DefaultTextLinkComponent(text string, link string) TextLinkComponent {
	return TextLinkComponent{
		Text:   text,
		Color:  TextColorAqua,
		Style:  TextLinkUnderlinedStyle,
		Action: OpenUrlAction(link),
	}
}

func (c TextLinkComponent) Bytes() []byte {
	w := NewWriterWithBuffer()
	w.WriteByte(0x0A)
	w.WriteStringEntry("text", c.Text)
	w.WriteStringEntry("color", c.Color)
	w.WriteByte(0x01)
	w.WriteString(c.Style)
	w.WriteByte(1)
	w.WriteByte(0x0A)
	w.WriteString("clickEvent")
	w.WriteStringEntry("action", c.Action.Action)
	w.WriteStringEntry("value", c.Action.Value)
	w.WriteByte(0x00)
	w.WriteByte(0x00)
	return w.Bytes()
}
