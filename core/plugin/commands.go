package plugin

import (
	"context"
	"fmt"

	"GoCraft/core/command"
)

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
			return commands.InvokeCommand(ctx, localExecutor, call.Sender, call.Args)
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
