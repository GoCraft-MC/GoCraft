package handler

import "sort"

type CommandPermission struct {
	Command        string `json:"command"`
	Node           string `json:"node"`
	DefaultAllowed bool   `json:"default_allowed"`
}

// CommandPermissions returns a stable snapshot for permission editors and
// diagnostics. Every registered command has exactly one permission node.
func (d *Dispatcher) CommandPermissions() []CommandPermission {
	d.mu.RLock()
	permissions := make([]CommandPermission, 0, len(d.cmds))
	for name, command := range d.cmds {
		permissions = append(permissions, CommandPermission{
			Command:        name,
			Node:           command.permission,
			DefaultAllowed: command.defaultAllow,
		})
	}
	d.mu.RUnlock()
	sort.Slice(permissions, func(i, j int) bool {
		return permissions[i].Command < permissions[j].Command
	})
	return permissions
}
