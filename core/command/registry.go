package command

import (
	"fmt"
	"sync"
)

type registration struct {
	root     Root
	source   Source
	handlers map[ExecID]Handler
}

// Registry owns command trees and their executor callbacks by source.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]registration
	version uint64
}

func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]registration)}
}

func (r *Registry) Register(source Source, root Root, handlers map[ExecID]Handler) error {
	if err := Validate(&root); err != nil {
		return err
	}
	key, err := sourceKey(source)
	if err != nil {
		return err
	}
	executors := make(map[ExecID]struct{})
	collectExecutors(root.Children, executors)
	for executor := range executors {
		if handlers[executor] == nil {
			return fmt.Errorf("command source %s: executor %d has no handler", key, executor)
		}
	}
	if len(executors) != len(handlers) {
		return fmt.Errorf("command source %s: handler count does not match the tree", key)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[key]; exists {
		return fmt.Errorf("command source %s is already registered", key)
	}
	for _, existing := range r.entries {
		if source.Kind != SourceCore && existing.source.Kind != SourceCore {
			continue
		}
		if conflict := rootConflict(root.Children, existing.root.Children); conflict != "" {
			return fmt.Errorf("command /%s conflicts with a core command", conflict)
		}
	}
	copyHandlers := make(map[ExecID]Handler, len(handlers))
	for executor, handler := range handlers {
		copyHandlers[executor] = handler
	}
	r.entries[key] = registration{root: root, source: source, handlers: copyHandlers}
	r.version++
	return nil
}

func (r *Registry) RevokeAll(pluginID string) {
	r.mu.Lock()
	if _, exists := r.entries[pluginID]; exists {
		delete(r.entries, pluginID)
		r.version++
	}
	r.mu.Unlock()
}

func (r *Registry) Version() uint64 {
	r.mu.RLock()
	version := r.version
	r.mu.RUnlock()
	return version
}
