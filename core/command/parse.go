package command

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

// ErrNoSuchCommand means the line names no command this snapshot holds. It is
// how a caller tells "not mine" from "yours, and wrong": the first lets the
// line fall through to whoever else dispatches commands, the second is a
// message for whoever typed it.
var ErrNoSuchCommand = errors.New("command: no such command")

// errNoMatch is the same distinction one node down, and never leaves this file.
// A sibling that does not match is not a failure — the next one may.
var errNoMatch = errors.New("command: node does not match")

// tickDuration is what a bare number means in a duration argument, matching how
// vanilla spells one: "40" is forty ticks, "40s" is forty seconds.
const tickDuration = 50 * time.Millisecond

// Resolvers supply the lookups an argument type needs from the running server.
//
// They are functions rather than a dependency on the game core because this
// package must stay edition-neutral and testable without a server: a parser
// test supplies three closures, not a world.
//
// A nil field means the host offers no lookup for that type, and an argument
// declaring it is refused rather than resolved to something invented.
type Resolvers struct {
	// Player resolves an online player by name.
	Player func(name string) (*player.Player, bool)

	// Block resolves a namespaced block identifier to its state.
	Block func(id string) (coreworld.Block, bool)

	// Item resolves a namespaced item identifier to a single-item stack.
	Item func(id string) (player.ItemStack, bool)
}

// Resolve walks one typed line through the commands this snapshot holds and
// reports which executor it reached with which arguments.
//
// It runs against a snapshot rather than the registry on purpose. A snapshot is
// already pruned to what this sender may use and already namespaced, so a
// branch guarded by a permission they lack is not merely refused — it is not
// there, and the line reads as an unknown command rather than as a denial. That
// is what keeps a plugin's admin commands from being discoverable by trying
// them, and it means namespacing needs no code here at all.
//
// The leading slash is optional: both editions hand this over with and without
// one depending on where the line came from.
func (s Snapshot) Resolve(line string, resolvers Resolvers) (ExecID, Values, error) {
	tokens := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "/"))
	if len(tokens) == 0 {
		return 0, nil, ErrNoSuchCommand
	}
	for _, child := range s.Root.Children {
		literal, ok := child.(Literal)
		if !ok || literal.Name != tokens[0] {
			continue
		}
		executor, values, err := descend(literal.Children, literal.Exec, tokens[1:], resolvers)
		if err != nil {
			return 0, nil, err
		}
		return executor, values, nil
	}
	return 0, nil, ErrNoSuchCommand
}

// descend consumes what is left of the line against one node's children.
//
// The values map is built at the leaf and filled on the way back out, so a
// branch that fails half way never has to undo what it wrote. Backtracking is
// then simply returning an error.
func descend(children []Node, executor ExecID, tokens []string, resolvers Resolvers) (ExecID, Values, error) {
	if len(tokens) == 0 {
		if executor != 0 {
			return executor, Values{}, nil
		}
		return 0, nil, fmt.Errorf("incomplete command, expected %s", expectation(children))
	}
	if len(children) == 0 {
		return 0, nil, fmt.Errorf("too many arguments: %q", strings.Join(tokens, " "))
	}

	// Literals first, then arguments in declaration order — the order the tree
	// was written in, which is the one an author expects to be tried.
	//
	// A sibling that simply does not match says nothing worth reporting. One
	// that matched and then failed does: it is the difference between telling
	// someone "unknown command" and telling them their price is out of range,
	// and the second is the only one they can act on. So the first real failure
	// is kept and reported if nothing else succeeds.
	var reported error
	for _, child := range children {
		reached, values, err := match(child, tokens, resolvers)
		if err == nil {
			return reached, values, nil
		}
		if reported == nil && !errors.Is(err, errNoMatch) {
			reported = err
		}
	}
	if reported != nil {
		return 0, nil, reported
	}
	return 0, nil, fmt.Errorf("unexpected %q, expected %s", tokens[0], expectation(children))
}

func match(node Node, tokens []string, resolvers Resolvers) (ExecID, Values, error) {
	switch typed := node.(type) {
	case Literal:
		if tokens[0] != typed.Name {
			return 0, nil, errNoMatch
		}
		return descend(typed.Children, typed.Exec, tokens[1:], resolvers)
	case Argument:
		consumed, value, err := parseArgument(typed, tokens, resolvers)
		if err != nil {
			return 0, nil, err
		}
		reached, values, err := descend(typed.Children, typed.Exec, tokens[consumed:], resolvers)
		if err != nil {
			return 0, nil, err
		}
		values[typed.Name] = value
		return reached, values, nil
	}
	return 0, nil, errNoMatch
}

// parseArgument reads one argument off the front of tokens and reports how many
// it consumed. Every failure it returns is a sentence for whoever typed the
// line, not a diagnostic: they are the only person who can fix it.
func parseArgument(argument Argument, tokens []string, resolvers Resolvers) (int, Value, error) {
	value := Value{Type: argument.Type}
	switch argument.Type {
	case ArgInteger:
		number, err := strconv.ParseInt(tokens[0], 10, 64)
		if err != nil {
			return 0, Value{}, fmt.Errorf("%s must be a whole number", argument.Name)
		}
		if argument.IntegerMin != nil && number < *argument.IntegerMin ||
			argument.IntegerMax != nil && number > *argument.IntegerMax {
			return 0, Value{}, fmt.Errorf("%s must be %s", argument.Name, integerRange(argument))
		}
		value.Integer = number
		return 1, value, nil

	case ArgDecimal:
		number, err := strconv.ParseFloat(tokens[0], 64)
		if err != nil {
			return 0, Value{}, fmt.Errorf("%s must be a number", argument.Name)
		}
		if argument.DecimalMin != nil && number < *argument.DecimalMin ||
			argument.DecimalMax != nil && number > *argument.DecimalMax {
			return 0, Value{}, fmt.Errorf("%s must be %s", argument.Name, decimalRange(argument))
		}
		value.Decimal = number
		return 1, value, nil

	case ArgString, ArgCustom:
		// A custom type is handed over as it was typed. Resolving it needs the
		// plugin's own resolver, which the ABI has no frame for yet, so the
		// honest thing is to pass the word through rather than guess at it.
		value.String = tokens[0]
		return 1, value, nil

	case ArgGreedy:
		// Rejoined on single spaces rather than sliced out of the original
		// line: the tree is what says an argument is greedy, and by the time we
		// know that the line has already been split. Runs of whitespace inside
		// a message are not worth carrying a second representation for.
		value.String = strings.Join(tokens, " ")
		return len(tokens), value, nil

	case ArgEnum:
		for _, allowed := range argument.Enum {
			if allowed == tokens[0] {
				value.String = allowed
				return 1, value, nil
			}
		}
		return 0, Value{}, fmt.Errorf("%s must be one of %s",
			argument.Name, strings.Join(argument.Enum, ", "))

	case ArgPlayer:
		if resolvers.Player == nil {
			return 0, Value{}, unsupported(argument)
		}
		found, ok := resolvers.Player(tokens[0])
		if !ok {
			return 0, Value{}, fmt.Errorf("no player named %s is online", tokens[0])
		}
		value.Player = found
		return 1, value, nil

	case ArgBlockPos:
		if len(tokens) < 3 {
			return 0, Value{}, fmt.Errorf("%s needs three coordinates", argument.Name)
		}
		var coordinates [3]int32
		for index, raw := range tokens[:3] {
			number, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return 0, Value{}, fmt.Errorf("%s must be three whole coordinates", argument.Name)
			}
			coordinates[index] = int32(number)
		}
		value.Position = spatial.BlockPos{X: coordinates[0], Y: coordinates[1], Z: coordinates[2]}
		return 3, value, nil

	case ArgBlockState:
		if resolvers.Block == nil {
			return 0, Value{}, unsupported(argument)
		}
		block, ok := resolvers.Block(tokens[0])
		if !ok {
			return 0, Value{}, fmt.Errorf("no block named %s", tokens[0])
		}
		value.Block = block
		return 1, value, nil

	case ArgItem:
		if resolvers.Item == nil {
			return 0, Value{}, unsupported(argument)
		}
		item, ok := resolvers.Item(tokens[0])
		if !ok {
			return 0, Value{}, fmt.Errorf("no item named %s", tokens[0])
		}
		value.Item = item
		return 1, value, nil

	case ArgDuration:
		duration, err := parseDuration(tokens[0])
		if err != nil {
			return 0, Value{}, fmt.Errorf("%s must be a duration such as 30s or 5m", argument.Name)
		}
		value.Duration = duration
		return 1, value, nil
	}
	return 0, Value{}, unsupported(argument)
}

// parseDuration reads the spellings a player already knows from vanilla: a bare
// number of ticks, an explicit tick suffix, or a Go duration.
func parseDuration(raw string) (time.Duration, error) {
	if ticks, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return positive(time.Duration(ticks) * tickDuration)
	}
	if ticks, err := strconv.ParseInt(strings.TrimSuffix(raw, "t"), 10, 64); strings.HasSuffix(raw, "t") && err == nil {
		return positive(time.Duration(ticks) * tickDuration)
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	return positive(parsed)
}

func positive(duration time.Duration) (time.Duration, error) {
	if duration < 0 {
		return 0, fmt.Errorf("duration is negative")
	}
	return duration, nil
}

// unsupported names the argument rather than its type number: whoever reads
// this is a player, and the type is the server's problem, not theirs.
func unsupported(argument Argument) error {
	return fmt.Errorf("%s uses an argument this server cannot read", argument.Name)
}

func integerRange(argument Argument) string {
	switch {
	case argument.IntegerMin != nil && argument.IntegerMax != nil:
		return fmt.Sprintf("between %d and %d", *argument.IntegerMin, *argument.IntegerMax)
	case argument.IntegerMin != nil:
		return fmt.Sprintf("at least %d", *argument.IntegerMin)
	default:
		return fmt.Sprintf("at most %d", *argument.IntegerMax)
	}
}

func decimalRange(argument Argument) string {
	switch {
	case argument.DecimalMin != nil && argument.DecimalMax != nil:
		return fmt.Sprintf("between %g and %g", *argument.DecimalMin, *argument.DecimalMax)
	case argument.DecimalMin != nil:
		return fmt.Sprintf("at least %g", *argument.DecimalMin)
	default:
		return fmt.Sprintf("at most %g", *argument.DecimalMax)
	}
}

// expectation lists what could come next, so an incomplete line says what is
// missing instead of only that something is.
func expectation(children []Node) string {
	names := make([]string, 0, len(children))
	for _, child := range children {
		switch typed := child.(type) {
		case Literal:
			names = append(names, typed.Name)
		case Argument:
			names = append(names, "<"+typed.Name+">")
		}
	}
	if len(names) == 0 {
		return "nothing"
	}
	return strings.Join(names, " | ")
}
