package gocraft

import (
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
)

// On registers a listener owned by this plugin.
func (e *Events) On(eventType string, handler EventHandler) error {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" || handler == nil {
		return fmt.Errorf("gocraft: event type and handler are required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.active {
		return fmt.Errorf("gocraft: event registry is disabled")
	}
	e.listeners[eventType] = append(e.listeners[eventType], handler)
	return nil
}

func (e *Events) OnPlayerJoin(handler func(*PlayerJoinEvent)) error {
	if handler == nil {
		return fmt.Errorf("gocraft: player join handler is required")
	}
	return e.On(EventPlayerJoin, func(event Event) { handler(event.(*PlayerJoinEvent)) })
}

func (e *Events) OnBlockBreak(handler func(*BlockBreakEvent)) error {
	if handler == nil {
		return fmt.Errorf("gocraft: block break handler is required")
	}
	return e.On(EventBlockBreak, func(event Event) { handler(event.(*BlockBreakEvent)) })
}

func (e *Events) registeredTypes() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	types := make([]string, 0, len(e.listeners))
	for eventType := range e.listeners {
		types = append(types, eventType)
	}
	sort.Strings(types)
	return types
}

func (e *Events) dispatch(event Event) {
	e.mu.RLock()
	if !e.active {
		e.mu.RUnlock()
		return
	}
	listeners := append([]EventHandler(nil), e.listeners[event.Type()]...)
	e.mu.RUnlock()
	for _, listener := range listeners {
		e.call(listener, event)
	}
}

func (e *Events) call(listener EventHandler, event Event) {
	defer func() {
		if recovered := recover(); recovered != nil {
			e.logger.Error("plugin event panicked", "event", event.Type(),
				"panic", recovered, "stack", string(debug.Stack()))
		}
	}()
	listener(event)
}

func (e *Events) clear() {
	e.mu.Lock()
	e.active = false
	e.listeners = make(map[string][]EventHandler)
	e.mu.Unlock()
}
