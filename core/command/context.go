package command

import (
	"context"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

// Value is one argument after host-side parsing and validation.
type Value struct {
	Type     ArgType
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
	if !ok || value.Type != ArgString && value.Type != ArgGreedy && value.Type != ArgEnum && value.Type != ArgCustom {
		return "", false
	}
	return value.String, true
}

func (v Values) Integer(name string) (int64, bool) {
	value, ok := v[name]
	if !ok || value.Type != ArgInteger {
		return 0, false
	}
	return value.Integer, true
}

func (v Values) Decimal(name string) (float64, bool) {
	value, ok := v[name]
	if !ok || value.Type != ArgDecimal {
		return 0, false
	}
	return value.Decimal, true
}

type Context struct {
	Sender Sender
	Args   Values
	Node   ExecID
}

type Handler func(context.Context, *Context) error
