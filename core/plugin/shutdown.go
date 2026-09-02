package plugin

import (
	"context"
	"errors"
	"fmt"
)

func (r *Registry) startRuntimes(ctx context.Context, bundles []Bundle) error {
	required, err := r.neededRuntimes(bundles)
	if err != nil {
		return err
	}
	for _, item := range required {
		if err := item.runtime.Start(ctx, r.host); err != nil {
			stopErr := r.stopRuntimes(ctx)
			return errors.Join(fmt.Errorf("start runtime %s: %w", item.name, err), stopErr)
		}
		r.mu.Lock()
		r.started = append(r.started, item.runtime)
		r.mu.Unlock()
	}
	return nil
}

// readyRuntimes ends the load phase for every runtime that has one.
//
// It runs once, after the last plugin is up, rather than once per plugin: a
// runtime told it is ready while the host is still loading would let its
// plugins act on a world the rest of them have not seen yet.
func (r *Registry) readyRuntimes(ctx context.Context, bundles []Bundle) error {
	required, err := r.neededRuntimes(bundles)
	if err != nil {
		return err
	}
	for _, item := range required {
		ready, ok := item.runtime.(ReadyRuntime)
		if !ok {
			continue
		}
		if err := ready.Ready(ctx); err != nil {
			return fmt.Errorf("ready runtime %s: %w", item.name, err)
		}
	}
	return nil
}

// Stop unloads plugins and runtimes in reverse load order.
func (r *Registry) Stop(ctx context.Context) error {
	r.mu.Lock()
	instances := append([]Instance(nil), r.loadOrder...)
	r.loadOrder = nil
	r.instances = make(map[string]Instance)
	r.mu.Unlock()
	var joined error
	for index := len(instances) - 1; index >= 0; index-- {
		manifest := instances[index].Manifest()
		r.bus.Detach(manifest.ID)
		r.commands.RevokeAll(manifest.ID)
		if err := instances[index].Unload(ctx); err != nil {
			joined = errors.Join(joined, fmt.Errorf("unload plugin %s: %w", manifest.ID, err))
		}
	}
	return errors.Join(joined, r.stopRuntimes(ctx))
}

func (r *Registry) stopRuntimes(ctx context.Context) error {
	r.mu.Lock()
	started := append([]Runtime(nil), r.started...)
	r.started = nil
	r.mu.Unlock()
	var joined error
	for index := len(started) - 1; index >= 0; index-- {
		if err := started[index].Stop(ctx); err != nil {
			joined = errors.Join(joined, fmt.Errorf("stop runtime %s: %w", started[index].Name(), err))
		}
	}
	return joined
}
