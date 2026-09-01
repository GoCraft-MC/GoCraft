package server

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"GoCraft/core/command"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

// defaultNamespace is what an unqualified identifier means, as everywhere else
// in the protocol: "stone" and "minecraft:stone" name the same block.
const defaultNamespace = "minecraft"

// runPluginCommand answers one typed line from the plugin command registry.
//
// It is the last link of the chain §07 describes and the only one that was
// missing: the tree comes from the bundle, the host parses against it, the
// arguments are resolved here, and the invocation crosses to whichever runtime
// loaded the plugin. Nothing below this knows which edition typed the line —
// the sender does, and it is the only thing that has to.
func (s *Server) runPluginCommand(sender *player.Player, line string) (bool, error) {
	if s.pluginRegistry == nil {
		return false, nil
	}
	handled, err := s.pluginRegistry.Commands().Execute(
		context.Background(), s.commandSenderFor(sender), line, s.commandResolvers())
	if err == nil {
		return handled, nil
	}
	// A refusal the sender caused reads as a sentence; anything else is the
	// server's fault and is logged rather than shown, because there is nothing
	// they could do about it.
	switch {
	case errors.Is(err, command.ErrPermission):
		return true, errors.New("You do not have permission to use this command")
	case errors.Is(err, command.ErrUnknownExecutor):
		slog.Warn("plugin command has no handler", "line", line)
		return true, errors.New("That command is not available right now")
	}
	return true, err
}

// commandResolvers supplies the lookups core/command cannot make for itself.
//
// Player is answered from the edition-neutral registry rather than from the
// Java session manager, so a Bedrock player is as nameable as a Java one.
//
// Block and item identifiers are normalised and accepted rather than checked
// against a registry: internal/gamedata exposes no lookup to check them with
// today. A handler therefore receives a well-formed identifier, not a proven
// one — worth tightening the moment that lookup exists.
func (s *Server) commandResolvers() command.Resolvers {
	return command.Resolvers{
		Player: func(name string) (*player.Player, bool) {
			var found *player.Player
			s.game.OnlinePlayers(func(candidate *player.Player) {
				if found == nil && strings.EqualFold(candidate.Username, name) {
					found = candidate
				}
			})
			return found, found != nil
		},
		Block: func(id string) (coreworld.Block, bool) {
			namespace, name, ok := splitIdentifier(id)
			if !ok {
				return coreworld.Block{}, false
			}
			return coreworld.Block{Namespace: namespace, Name: name}, true
		},
		Item: func(id string) (player.ItemStack, bool) {
			namespace, name, ok := splitIdentifier(id)
			if !ok {
				return player.ItemStack{}, false
			}
			return player.ItemStack{ItemID: namespace + ":" + name, Count: 1}, true
		},
	}
}

// splitIdentifier reads a resource location, defaulting the namespace. It
// refuses anything that is not one rather than passing a typo through as a
// block nobody has ever heard of.
func splitIdentifier(id string) (namespace, name string, ok bool) {
	namespace, name = defaultNamespace, strings.ToLower(strings.TrimSpace(id))
	if separator := strings.IndexByte(name, ':'); separator >= 0 {
		namespace, name = name[:separator], name[separator+1:]
	}
	if !validIdentifierPart(namespace) || !validIdentifierPart(name) {
		return "", "", false
	}
	return namespace, name, true
}

func validIdentifierPart(part string) bool {
	if part == "" {
		return false
	}
	for _, character := range part {
		switch {
		case character >= 'a' && character <= 'z',
			character >= '0' && character <= '9',
			character == '_', character == '-', character == '.', character == '/':
		default:
			return false
		}
	}
	return true
}
