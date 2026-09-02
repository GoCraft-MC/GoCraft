package dispatch

import (
	"context"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"

	"github.com/GoCraft-MC/gocraft-abi/command"
)

// Value is one argument after host-side parsing and validation.
type Value struct {
	Type     command.ArgType
	Integer  int64
	Decimal  float64
	String   string
	Player   *player.Player
	Position spatial.BlockPos
	Block    coreworld.Block
	Item     player.ItemStack
	Duration time.Duration
}

type Values map[string]Value

func (v Values) Get(name string) (Value, bool) {
	value, ok := v[name]
	return value, ok
}

func (v Values) String(name string) (string, bool) {
	value, ok := v[name]
	if !ok || value.Type != command.ArgString && value.Type != command.ArgGreedy && value.Type != command.ArgEnum && value.Type != command.ArgCustom {
		return "", false
	}
	return value.String, true
}

func (v Values) Integer(name string) (int64, bool) {
	value, ok := v[name]
	if !ok || value.Type != command.ArgInteger {
		return 0, false
	}
	return value.Integer, true
}

func (v Values) Decimal(name string) (float64, bool) {
	value, ok := v[name]
	if !ok || value.Type != command.ArgDecimal {
		return 0, false
	}
	return value.Decimal, true
}

type Context struct {
	Sender Sender
	Args   Values
	Node   command.ExecID

	// Raw is what followed the command name, split on whitespace.
	//
	// Args is the contract: the host parsed the line against the tree and every
	// argument here is of the type its node declared. Raw exists beside it for
	// a handler written before that was true, so the built-ins can move onto
	// this registry one at a time instead of in one commit that touches fifty
	// commands at once. Empty when an executor was invoked without a line —
	// a plugin command called by id has no tokens to report.
	Raw []string
}

type Handler func(context.Context, *Context) error
