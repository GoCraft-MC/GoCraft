// Package dispatch turns a typed command line into a call.
//
// The tree it walks is a format and lives in gocraft-abi; what is here is
// everything that needs a running world to mean anything — who is speaking,
// what they may see, which handler answers, and how a plugin's tree is grafted
// onto the server's without either being able to take the other's names.
package dispatch

import "GoCraft/core/player"

// Sender is implemented by Java, Bedrock, and console command sources.
type Sender interface {
	Name() string
	UUID() [16]byte
	Has(permission string) bool
	SendMessage(message string) error
	Player() (*player.Player, bool)
}

type SourceKind uint8

const (
	SourceCore SourceKind = iota
	SourcePlugin
)

// Source is who registered a command, which decides what it may be called and
// what happens when two of them want the same name.
type Source struct {
	Kind     SourceKind
	PluginID string
}
