package gocraft

import (
	"time"

	abi "GoCraft/abi/v1"
)

type CommandValueKind uint8

const (
	CommandInteger CommandValueKind = iota + 1
	CommandDecimal
	CommandString
	CommandGreedy
	CommandPlayer
	CommandBlockPos
	CommandBlockState
	CommandItem
	CommandDuration
	CommandEnum
	CommandCustom
)

// CommandHandler handles one executor declared in the bundle's command tree.
type CommandHandler func(*CommandContext) error

// CommandContext contains host-validated command arguments.
type CommandContext struct {
	Sender     *Player
	SenderName string
	Args       CommandValues
	replies    []string

	// permissions carries what the host resolved before sending the
	// invocation. The ABI has no message for asking afterwards, so this is the
	// only answer available — see Can.
	permissions abi.CommandSender
}

// Can reports whether the sender holds a permission node.
//
// Answered from what the host already resolved, not by asking it: the ABI has
// no message for asking, and one that existed would be a round trip inside a
// command somebody is waiting on. A node this plugin's manifest never declared
// reads false, which is a manifest bug rather than a denial.
func (c *CommandContext) Can(node string) bool {
	return c.permissions.Allowed(node)
}

// Reply sends text to the player who invoked the command.
func (c *CommandContext) Reply(message string) {
	if message != "" {
		c.replies = append(c.replies, message)
	}
}

// CommandValue is one typed command argument.
type CommandValue struct {
	Kind     CommandValueKind
	Integer  int64
	Decimal  float64
	Text     string
	Player   *Player
	Position BlockPos
	Block    Block
	Item     Item
	Duration time.Duration
}

// Item is the ItemRef vocabulary type: what a command's item argument carries.
//
// Not a whole stack. The argument is parsed from an id and a count, so
// enchantments and component data would arrive empty on every invocation.
type Item struct {
	ID     string
	Count  int64
	Damage int64
}

// CommandValues is keyed by the argument name from the command tree.
type CommandValues map[string]CommandValue

func (v CommandValues) String(name string) (string, bool) {
	value, ok := v[name]
	if !ok || value.Text == "" {
		return "", false
	}
	return value.Text, true
}

func (v CommandValues) Integer(name string) (int64, bool) {
	value, ok := v[name]
	return value.Integer, ok
}
