package command

import "sort"

// Snapshot is an immutable command view for one sender.
type Snapshot struct {
	Root    Root
	Version uint64
}

func (r *Registry) Snapshot(sender Sender) Snapshot {
	r.mu.RLock()
	entries := make(map[string]registration, len(r.entries))
	for key, entry := range r.entries {
		entries[key] = entry
	}
	version := r.version
	r.mu.RUnlock()

	keys := make([]string, 0, len(entries))
	pluginClaims := make(map[string]int)
	for key, entry := range entries {
		keys = append(keys, key)
		if entry.source.Kind == SourcePlugin {
			for _, node := range entry.root.Children {
				pluginClaims[node.(Literal).Name]++
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		left, right := entries[keys[i]].source, entries[keys[j]].source
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return keys[i] < keys[j]
	})

	root := Root{}
	for _, key := range keys {
		entry := entries[key]
		for _, node := range entry.root.Children {
			literal := node.(Literal)
			if entry.source.Kind == SourcePlugin && pluginClaims[literal.Name] > 1 {
				literal.Name = key + ":" + literal.Name
			}
			if visible, ok := visibleNode(literal, sender); ok {
				root.Children = append(root.Children, visible)
			}
		}
	}
	return Snapshot{Root: root, Version: version}
}

func visibleNode(node Node, sender Sender) (Node, bool) {
	switch typed := node.(type) {
	case Literal:
		if typed.Permission != "" && (sender == nil || !sender.Has(typed.Permission)) {
			return nil, false
		}
		typed.Children = visibleNodes(typed.Children, sender)
		if typed.Exec == 0 && len(typed.Children) == 0 {
			return nil, false
		}
		return typed, true
	case Argument:
		typed = cloneArgument(typed)
		typed.Children = visibleNodes(typed.Children, sender)
		if typed.Exec == 0 && len(typed.Children) == 0 {
			return nil, false
		}
		return typed, true
	default:
		return nil, false
	}
}

func visibleNodes(nodes []Node, sender Sender) []Node {
	visible := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if cloned, ok := visibleNode(node, sender); ok {
			visible = append(visible, cloned)
		}
	}
	return visible
}
