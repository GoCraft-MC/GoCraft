package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"GoCraft/core/command"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

// EffectMessage delivers one line to whoever an effect names. It is the only
// host call the server implements so far, and both the queue and the command
// path have to agree on the string.
const EffectMessage = "chat.message"

// deliverCommandEffects performs what a command handler asked for.
//
// Almost everything is queued for the tick, like a verdict's effects: a handler
// runs on another thread, in another process, and touching world state from
// there would race the simulation reading it.
//
// A reply to a sender with no player is the exception. It travels as a
// chat.message carrying an empty PlayerRef, which the tick drops on purpose —
// an event's empty ref means a block broken by a piston, and nobody to tell.
// A command always has somebody to tell, even from the console, so it is
// answered through the sender directly. Writing a line to a console is not
// world state and needs no tick.
//
// One effect failing does not stop the others: they are independent requests,
// and dropping the tail because the head was malformed would turn one bad call
// into a handler that appears to have done nothing.
func (r *Registry) deliverCommandEffects(pluginID string, executor command.ExecID,
	sender command.Sender, effects []abi.HostCall) {
	for _, effect := range effects {
		if text, ok := consoleReply(effect, sender); ok {
			if err := sender.SendMessage(text); err != nil {
				slog.Error("reply to a command sender",
					"plugin", pluginID, "executor", executor, "err", err)
			}
			continue
		}
		if err := r.bus.host.Enqueue(effect); err != nil {
			slog.Error("queue plugin command effect",
				"plugin", pluginID, "executor", executor, "effect", effect.Type, "err", err)
		}
	}
}

// consoleReply reports whether an effect is a reply the tick would silently
// drop, and returns the line to send instead.
func consoleReply(effect abi.HostCall, sender command.Sender) (string, bool) {
	if effect.Type != EffectMessage || sender == nil || len(effect.Fields) != 2 {
		return "", false
	}
	if _, addressed := PlayerUUIDFrom(effect.Fields[0]); addressed {
		return "", false
	}
	return TextFrom(effect.Fields[1])
}

func (r *Registry) registerBundleCommands(bundles []Bundle) error {
	var registered []string
	for _, bundle := range bundles {
		if bundle.Commands == nil {
			continue
		}
		handlers := r.commandHandlers(bundle.Manifest.ID, *bundle.Commands)
		source := command.Source{Kind: command.SourcePlugin, PluginID: bundle.Manifest.ID}
		if err := r.commands.Register(source, *bundle.Commands, handlers); err != nil {
			for _, pluginID := range registered {
				r.commands.RevokeAll(pluginID)
			}
			return fmt.Errorf("plugin %s commands: %w", bundle.Manifest.ID, err)
		}
		registered = append(registered, bundle.Manifest.ID)
	}
	return nil
}

func (r *Registry) commandHandlers(pluginID string, root command.Root) map[command.ExecID]command.Handler {
	handlers := make(map[command.ExecID]command.Handler)
	for _, executor := range command.Executors(root) {
		localExecutor := executor
		handlers[executor] = func(ctx context.Context, call *command.Context) error {
			instance, loaded := r.Instance(pluginID)
			if !loaded {
				return fmt.Errorf("plugin %s is not loaded", pluginID)
			}
			commands, ok := instance.(CommandInstance)
			if !ok {
				return fmt.Errorf("plugin %s runtime does not support commands", pluginID)
			}
			result, err := commands.InvokeCommand(ctx, localExecutor, call.Sender, call.Args)
			if err != nil {
				return err
			}
			// Queued whether or not the handler reported a failure. A handler
			// that refused and said why has already asked for that message, and
			// dropping it because it also returned an error would leave the
			// refusal silent — the sender would see nothing at all.
			r.deliverCommandEffects(pluginID, localExecutor, call.Sender, result.Effects)
			if result.Error != "" {
				return errors.New(result.Error)
			}
			return nil
		}
	}
	return handlers
}

func (r *Registry) revokeBundleCommands(bundles []Bundle) {
	for _, bundle := range bundles {
		if bundle.Commands != nil {
			r.commands.RevokeAll(bundle.Manifest.ID)
		}
	}
}
