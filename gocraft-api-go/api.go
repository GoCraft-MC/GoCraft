// Package gocraft is the public API used by native Go plugins.
package gocraft

import "log/slog"

// CurrentVersion is the Go Plugin API version supported by this server.
const CurrentVersion uint32 = 1

// Metadata identifies a compiled plugin. The bundle manifest remains the
// host's authoritative copy and is checked before plugin code starts.
type Metadata struct {
	ID         string
	Version    string
	APIVersion uint32
}

// Plugin is the lifecycle implemented by a native Go plugin.
type Plugin interface {
	OnLoad(Context) error
	OnEnable() error
	OnDisable() error
}

// Context exposes resources owned by one plugin.
type Context interface {
	Metadata() Metadata
	DataDirectory() string
	Logger() *slog.Logger
	Events() *Events
	Commands() *Commands
	Scheduler() *Scheduler
}

// Player is a protocol-independent player reference.
type Player struct {
	UUID     [16]byte
	Username string
	Edition  string
}

// BlockPos is an integer position in a world.
type BlockPos struct {
	X int32
	Y int32
	Z int32
}

// Block is a namespaced block state snapshot.
type Block struct {
	ID         string
	Properties map[string]string
}
