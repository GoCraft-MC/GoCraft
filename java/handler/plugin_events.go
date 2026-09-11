package handler

import (
	"strings"
	"unicode/utf8"

	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
)

func (d *Dispatcher) SetEventBus(bus *coreplugin.Bus) {
	d.mu.Lock()
	d.pluginEvents = bus
	d.mu.Unlock()
}

func (d *Dispatcher) EventBus() *coreplugin.Bus {
	if d == nil {
		return nil
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.pluginEvents
}

// FilterPlayerChat runs once, before either edition formats the message.
// Rewritten text is never fed back into command dispatch.
func (d *Dispatcher) FilterPlayerChat(p *player.Player, message *string) bool {
	if events := d.EventBus(); events != nil && p != nil && !events.EmitPlayerChat(p, message) {
		return false
	}
	*message = strings.TrimSpace(*message)
	return *message != "" && validEventText(*message)
}

func validEventText(text string) bool {
	return utf8.ValidString(text) && utf8.RuneCountInString(text) <= maxChatLength &&
		!strings.ContainsAny(text, "\x00\r\n")
}
