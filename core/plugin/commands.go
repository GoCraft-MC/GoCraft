package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	abi "GoCraft/abi/v1"
	"GoCraft/core/command"
)

// enqueueCommandEffects queues what a command handler asked the world to do.
//
// One effect failing does not stop the others: they are independent requests,
// and dropping the tail because the head was malformed would turn one bad call
// into a handler that appears to have done nothing.
func (r *Registry) enqueueCommandEffects(pluginID string, executor command.ExecID, effects []abi.HostCall) {
	for _, effect := range effects {
		if err := r.bus.host.Enqueue(effect); err != nil {
			slog.Error("queue plugin command effect",
				"plugin", pluginID, "executor", executor, "effect", effect.Type, "err", err)
		}
	}
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
			r.enqueueCommandEffects(pluginID, localExecutor, result.Effects)
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
