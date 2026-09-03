package handler

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"GoCraft/core/dispatch"
	"github.com/GoCraft-MC/gocraft-abi/command"
)

// The built-in command tree, as data.
//
// It used to exist twice: once as a Brigadier graph built by hand for Java
// clients, and once as a name-keyed map the dispatcher routed on. Neither knew
// about the other, so a command could be dispatchable and unadvertised, or
// advertised and gone. §18 collapses them into one neutral tree that both
// editions render and that plugin registration is checked against.
//
// What has not moved yet is argument parsing. A handler still reads
// Context.Raw, because §18 extracts the command system rather than rewriting
// fifty handlers in one commit — the tree describes what a command accepts, and
// each handler moves onto typed values on its own schedule.

// builtinCommand is what one command advertises beyond its name.
type builtinCommand struct {
	// executable marks a command that runs with no arguments at all. /kill does
	// and /gamemode does not, and a client is told which.
	executable bool
	children   []command.Node
}

// literal is a node that runs, since every executable node in this tree reaches
// the same handler: the command's name is what the dispatcher routes on.
func literal(name string, children ...command.Node) command.Node {
	return command.Literal{Name: name, Exec: builtinExec, Children: children}
}

// branch is a node that only leads somewhere.
func branch(name string, children ...command.Node) command.Node {
	return command.Literal{Name: name, Children: children}
}

func argument(name string, kind command.ArgType, children ...command.Node) command.Node {
	return command.Argument{Name: name, Type: kind, Exec: builtinExec, Children: children}
}

// argumentBranch is an argument that must be followed by more.
func argumentBranch(name string, kind command.ArgType, children ...command.Node) command.Node {
	return command.Argument{Name: name, Type: kind, Children: children}
}

func integerArgument(name string, minimum, maximum int64, children ...command.Node) command.Node {
	return command.Argument{
		Name: name, Type: command.ArgInteger, Exec: builtinExec, Children: children,
		IntegerMin: &minimum, IntegerMax: &maximum,
	}
}

func decimalArgument(name string, minimum, maximum float64) command.Node {
	return command.Argument{
		Name: name, Type: command.ArgDecimal, Exec: builtinExec,
		DecimalMin: &minimum, DecimalMax: &maximum,
	}
}

// builtinExec is the local executor every node of this tree carries.
//
// One id for the whole tree, because the dispatcher routes on the command name
// and hands the handler its raw tokens: which node a line reached does not
// decide what runs, and pretending otherwise would mean inventing an id per
// node that nothing reads. Each command gets its own id when the tree is built.
const builtinExec command.ExecID = 1

// freeArguments is what a command whose arguments are not modelled advertises:
// the rest of the line, unparsed. It is the shape most of these already had.
func freeArguments() []command.Node {
	return []command.Node{argument("arguments", command.ArgGreedy)}
}

func playerTarget() command.Node { return argument("player", command.ArgPlayer) }

// builtinArguments describes the commands whose arguments are worth completing.
// Anything registered and absent from this table advertises freeArguments.
func builtinArguments() map[string]builtinCommand {
	modes := func() []command.Node {
		return []command.Node{
			literal("survival"), literal("creative"), literal("adventure"), literal("spectator"),
		}
	}
	speed := func() builtinCommand {
		return builtinCommand{children: []command.Node{
			decimalArgument("value", 0.001, 1), literal("reset"),
		}}
	}
	effectTarget := func() []command.Node {
		effects := make([]command.Node, 0, len(potionEffectNames))
		for _, effect := range potionEffectNames {
			effects = append(effects, branch(effect, integerArgument("seconds", 1, 1_000_000)))
		}
		return []command.Node{argumentBranch("player", command.ArgPlayer, effects...)}
	}

	table := map[string]builtinCommand{
		"gamemode": {children: modes()},
		"gm":       {children: modes()},
		"gocraft": {executable: true, children: []command.Node{
			literal("peditor"),
			branch("applyedits", argument("link", command.ArgGreedy)),
		}},
		"tp": {children: []command.Node{
			argumentBranch("x", command.ArgDecimal,
				argumentBranch("y", command.ArgDecimal, argument("z", command.ArgDecimal))),
			playerTarget(),
		}},
		"world": {executable: true, children: []command.Node{
			literal("overworld"), literal("nether"), literal("end"),
		}},
		"tphere": {children: []command.Node{playerTarget()}},
		"give": {children: []command.Node{
			argumentBranch("player", command.ArgPlayer,
				argument("item", command.ArgItem, integerArgument("count", 1, 64))),
		}},
		"get": {children: []command.Node{
			argument("item", command.ArgItem, integerArgument("count", 1, 64)),
		}},
		"kick": {children: []command.Node{
			argument("player", command.ArgPlayer, argument("reason", command.ArgGreedy)),
		}},
		"kill":  {executable: true, children: []command.Node{playerTarget()}},
		"op":    {children: []command.Node{playerTarget()}},
		"god":   {executable: true, children: []command.Node{playerTarget()}},
		"ungod": {executable: true, children: []command.Node{playerTarget()}},
		"heal":  {executable: true, children: []command.Node{playerTarget()}},
		"time": {executable: true, children: []command.Node{
			literal("day"), literal("night"),
			branch("set", integerArgument("value", 0, 23999)),
		}},
		"whitelist": {executable: true, children: []command.Node{
			literal("on"), literal("off"), literal("list"),
			branch("add", argument("player", command.ArgString)),
			branch("remove", argument("player", command.ArgString)),
		}},
		"potioneffect": {children: effectTarget()},
		"effect":       {children: effectTarget()},
	}
	for _, name := range []string{"walkspeed", "walkspeen", "flyspeed", "flyyspeed"} {
		table[name] = speed()
	}

	locations := make([]command.Node, 0, len(locatableTargets))
	for _, target := range locatableTargets {
		locations = append(locations, literal(target))
	}
	table["locate"] = builtinCommand{children: locations}

	professions := make([]command.Node, 0, len(villagerProfessionNames))
	for _, profession := range villagerProfessionNames {
		professions = append(professions, literal(profession))
	}
	mobs := make([]command.Node, 0, len(summonableMobNames))
	for _, mob := range summonableMobNames {
		if mob == "villager" {
			mobs = append(mobs, command.Literal{Name: mob, Exec: builtinExec, Children: professions})
			continue
		}
		mobs = append(mobs, literal(mob))
	}
	table["summon"] = builtinCommand{children: mobs}

	// Advertised as a bare name, with nothing after it.
	for _, name := range []string{
		"help", "list", "xyz", "version", "ver", "fly",
		"timings", "tps", "mspt", "spawn", "setspawn",
	} {
		table[name] = builtinCommand{executable: true}
	}
	return table
}

// commandTree renders every registered built-in as one tree.
//
// Every command appears, including the ones the hand-built graph never
// mentioned: a command that is dispatchable and unadvertised is one a player
// can only find by being told about it, which is not a feature.
func (d *Dispatcher) commandTree() (command.Root, map[command.ExecID]dispatch.Handler) {
	d.mu.RLock()
	names := make([]string, 0, len(d.cmds))
	registered := make(map[string]registeredCommand, len(d.cmds))
	for name, entry := range d.cmds {
		names = append(names, name)
		registered[name] = entry
	}
	d.mu.RUnlock()
	sort.Strings(names)

	table := builtinArguments()
	root := command.Root{}
	handlers := make(map[command.ExecID]dispatch.Handler, len(names))
	for index, name := range names {
		shape, described := table[name]
		children := shape.children
		if !described {
			children = freeArguments()
		}
		executor := command.ExecID(index + 1)
		node := command.Literal{
			Name: name, Permission: registered[name].permission,
			Children: retarget(children, executor),
		}
		if shape.executable || !described || len(children) == 0 {
			node.Exec = executor
		}
		root.Children = append(root.Children, node)
		handlers[executor] = d.builtinHandler(name)
	}
	return root, handlers
}

// retarget stamps one command's executor onto every node the table built with
// the placeholder. The table is written once and read per command, so the id
// cannot be baked into it.
func retarget(nodes []command.Node, executor command.ExecID) []command.Node {
	out := make([]command.Node, 0, len(nodes))
	for _, node := range nodes {
		switch typed := node.(type) {
		case command.Literal:
			if typed.Exec != 0 {
				typed.Exec = executor
			}
			typed.Children = retarget(typed.Children, executor)
			out = append(out, typed)
		case command.Argument:
			if typed.Exec != 0 {
				typed.Exec = executor
			}
			typed.Children = retarget(typed.Children, executor)
			out = append(out, typed)
		}
	}
	return out
}

// builtinHandler is what the registry holds for a built-in.
//
// It is never reached. Dispatch answers for every name in this tree before it
// offers a line to any other registry, and a name is in this tree precisely
// because Dispatch has a handler for it. What the entry buys is the two things
// only a registered tree can: a client graph rendered from the same structure
// both editions share, and a plugin being refused a core command name at load
// rather than shadowing /tp until somebody notices.
//
// The handler runs when that stops being true, and says so rather than
// pretending a command took no arguments.
func (d *Dispatcher) builtinHandler(name string) dispatch.Handler {
	return func(context.Context, *dispatch.Context) error {
		return fmt.Errorf("/%s is dispatched by the server, not through the registry", name)
	}
}

// SetCommandRegistry gives the dispatcher the registry to publish its tree to.
//
// Publishing is not optional once set: every later registration republishes, so
// a command added while players are online reaches them by the same path a
// plugin's does.
func (d *Dispatcher) SetCommandRegistry(registry *dispatch.Registry) {
	d.mu.Lock()
	d.registry = registry
	d.mu.Unlock()
	d.publishTree()
}

// publishTree republishes the built-ins. A failure is logged rather than
// returned: it means the tree this file builds is malformed, which is a bug
// here and not something a caller registering a command can act on.
func (d *Dispatcher) publishTree() {
	d.mu.RLock()
	registry := d.registry
	d.mu.RUnlock()
	if registry == nil {
		return
	}
	root, handlers := d.commandTree()
	if len(root.Children) == 0 {
		return
	}
	if err := registry.Replace(dispatch.Source{Kind: dispatch.SourceCore}, root, handlers); err != nil {
		slog.Error("publishing the built-in command tree failed", "err", err)
	}
}
