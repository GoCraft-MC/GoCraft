// Package command defines the edition-neutral command tree and dispatcher.
package command

import "GoCraft/core/player"

// Sender is implemented by Java, Bedrock, and console command sources.
type Sender interface {
	Name() string
	UUID() [16]byte
	Has(permission string) bool
	SendMessage(message string) error
	Player() (*player.Player, bool)
}

type ExecID uint32

type ArgType uint8

const (
	ArgInteger ArgType = iota + 1
	ArgDecimal
	ArgString
	ArgGreedy
	ArgPlayer
	ArgBlockPos
	ArgBlockState
	ArgItem
	ArgDuration
	ArgEnum
	ArgCustom
)

// Node is sealed so every renderer sees the complete command IR.
type Node interface{ isNode() }

type Root struct {
	Children []Node
}

func (Root) isNode() {}

type Literal struct {
	Name       string
	Permission string
	Children   []Node
	Exec       ExecID
}

func (Literal) isNode() {}

type Argument struct {
	Name       string
	Type       ArgType
	Enum       []string
	CustomType string
	IntegerMin *int64
	IntegerMax *int64
	DecimalMin *float64
	DecimalMax *float64
	Children   []Node
	Exec       ExecID
}

func (Argument) isNode() {}

type SourceKind uint8

const (
	SourceCore SourceKind = iota
	SourcePlugin
)

type Source struct {
	Kind     SourceKind
	PluginID string
}
