// Package plugin owns plugin discovery, lifecycle, and event dispatch.
package plugin

import "GoCraft/core/command"

// Priority orders subscribers from earliest to latest.
type Priority int8

const (
	PriorityLowest Priority = iota
	PriorityLow
	PriorityNormal
	PriorityHigh
	PriorityHighest
	PriorityMonitor
)

// Subscription describes one event declared by a plugin manifest.
type Subscription struct {
	Event    string
	Priority Priority
}

// Manifest contains everything the host needs before plugin code starts.
type Manifest struct {
	ID            string
	Version       string
	APIVersion    uint32
	Runtime       string
	Entry         string
	CommandTree   string
	Permissions   []string
	Subscriptions []Subscription
}

// Bundle is one validated .gcpkg archive and its generated manifest.
type Bundle struct {
	Path     string
	Manifest Manifest
	Commands *command.Root
}
