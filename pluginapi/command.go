package pluginapi

import "time"

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
	Item     string
	Duration time.Duration
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
