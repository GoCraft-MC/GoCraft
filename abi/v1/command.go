package abi

// CommandArgumentType is what the host parsed an argument as.
//
// The numbers match CommandArgumentType in commands.proto so the transport
// converts with a switch over known values rather than a cast — a cast would
// turn a value from a newer runtime into whichever constant happened to share
// its number, silently.
type CommandArgumentType uint8

const (
	CommandArgumentInvalid CommandArgumentType = iota
	CommandArgumentInteger
	CommandArgumentDecimal
	CommandArgumentString
	CommandArgumentGreedy
	CommandArgumentPlayer
	CommandArgumentBlockPos
	CommandArgumentBlockState
	CommandArgumentItem
	CommandArgumentDuration
	CommandArgumentEnum
	CommandArgumentCustom
)

// CommandArgument is one argument the host already parsed and validated.
//
// Named, where an event's fields are positional: see CommandArgument in
// envelope.proto for why the two differ.
type CommandArgument struct {
	Name  string
	Type  CommandArgumentType
	Value Value
}

// CommandSender is whoever typed the command.
type CommandSender struct {
	// The PlayerRef vocabulary type. An empty list means there is no player
	// behind this — the console, or a command block.
	Player Value
	Name   string
	// [node, allowed] pairs, resolved before the invocation is sent because the
	// ABI has no message for asking.
	Permissions []Value
}

// CommandInvocation is one command executor and everything it needs to run.
//
// It carries no plugin id: like Event, the recipient is a separate argument to
// the call, because the host decides who receives what.
type CommandInvocation struct {
	Executor  uint32
	Sender    CommandSender
	Arguments []CommandArgument
}

// CommandResult is what a runtime answers to one invocation.
type CommandResult struct {
	// Empty means the command succeeded. Anything else is meant for a person.
	Error string
	// Everything the handler asked the world to do, replying to the sender
	// included. Queued by the host and applied by the tick, exactly as a
	// verdict's effects are.
	Effects []HostCall
}

// Allowed reports whether the sender holds a permission node.
//
// It reads the injected pairs rather than asking the server, which is the whole
// point of injecting them. A node the plugin's manifest never declared was
// never resolved, so it reads false — the same answer a runtime gives for an
// event, and for the same reason.
func (s CommandSender) Allowed(node string) bool {
	for _, pair := range s.Permissions {
		if pair.Kind != ValueList || len(pair.List) != 2 {
			continue
		}
		name, allowed := pair.List[0], pair.List[1]
		if name.Kind == ValueString && name.String == node && allowed.Kind == ValueBool {
			return allowed.Bool
		}
	}
	return false
}
