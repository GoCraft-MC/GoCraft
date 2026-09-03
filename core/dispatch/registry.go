package dispatch

import (
	"fmt"
	"sync"

	"github.com/GoCraft-MC/gocraft-abi/command"
)

type registration struct {
	root      command.Root
	source    Source
	executors []command.ExecID
}

type registeredHandler struct {
	source  Source
	handler Handler
}

// Registry owns command trees and their executor callbacks by source.
type Registry struct {
	mu       sync.RWMutex
	entries  map[string]registration
	handlers map[command.ExecID]registeredHandler
	nextExec command.ExecID
	version  uint64
}

func NewRegistry() *Registry {
	return &Registry{
		entries:  make(map[string]registration),
		handlers: make(map[command.ExecID]registeredHandler),
	}
}

// Replace installs a source's tree whether or not it already had one.
//
// Register refuses a second registration because two plugins claiming one id is
// a mistake. The built-ins are the case where it is not: they are declared as
// they are registered, so the tree grows through startup and every growth is a
// new version for clients already connected. Replacing drops the previous
// executors first, so a command that went away takes its handler with it.
func (r *Registry) Replace(source Source, root command.Root, handlers map[command.ExecID]Handler) error {
	if err := command.Validate(&root); err != nil {
		return err
	}
	key, err := sourceKey(source)
	if err != nil {
		return err
	}
	executors := command.Executors(root)
	for _, executor := range executors {
		if handlers[executor] == nil {
			return fmt.Errorf("command source %s: executor %d has no handler", key, executor)
		}
	}
	if len(executors) != len(handlers) {
		return fmt.Errorf("command source %s: handler count does not match the tree", key)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registerLocked(key, source, root, handlers, true)
}

func (r *Registry) Register(source Source, root command.Root, handlers map[command.ExecID]Handler) error {
	if err := command.Validate(&root); err != nil {
		return err
	}
	key, err := sourceKey(source)
	if err != nil {
		return err
	}
	executors := command.Executors(root)
	for _, executor := range executors {
		if handlers[executor] == nil {
			return fmt.Errorf("command source %s: executor %d has no handler", key, executor)
		}
	}
	if len(executors) != len(handlers) {
		return fmt.Errorf("command source %s: handler count does not match the tree", key)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.registerLocked(key, source, root, handlers, false)
}

func (r *Registry) registerLocked(key string, source Source, root command.Root, handlers map[command.ExecID]Handler, replace bool) error {
	if _, exists := r.entries[key]; exists && !replace {
		return fmt.Errorf("command source %s is already registered", key)
	}
	for existingKey, existing := range r.entries {
		if replace && existingKey == key {
			continue
		}
		if source.Kind != SourceCore && existing.source.Kind != SourceCore {
			continue
		}
		if conflict := rootConflict(root.Children, existing.root.Children); conflict != "" {
			return fmt.Errorf("command /%s conflicts with a core command", conflict)
		}
	}
	if previous, exists := r.entries[key]; exists {
		for _, executor := range previous.executors {
			delete(r.handlers, executor)
		}
		delete(r.entries, key)
	}
	root, remapped := remapRoot(root, func() command.ExecID {
		r.nextExec++
		return r.nextExec
	})
	globalIDs := make([]command.ExecID, 0, len(remapped))
	for local, global := range remapped {
		globalIDs = append(globalIDs, global)
		r.handlers[global] = registeredHandler{source: source, handler: handlers[local]}
	}
	r.entries[key] = registration{root: root, source: source, executors: globalIDs}
	r.version++
	return nil
}

func (r *Registry) RevokeAll(pluginID string) {
	r.mu.Lock()
	entry, exists := r.entries[pluginID]
	if exists && entry.source.Kind == SourcePlugin {
		for _, executor := range entry.executors {
			delete(r.handlers, executor)
		}
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
