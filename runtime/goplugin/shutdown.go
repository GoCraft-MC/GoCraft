package goplugin

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

func (r *Runtime) Ready(context.Context) error {
	instances := r.snapshot()
	for _, instance := range instances {
		if err := instance.supervisor.Ready(); err != nil {
			return fmt.Errorf("go runtime: ready plugin %s: %w", instance.manifest.ID, err)
		}
	}
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	instances := orderedInstances(r.instances)
	r.instances = make(map[string]*Instance)
	r.started = false
	r.mu.Unlock()
	var joined error
	for index := len(instances) - 1; index >= 0; index-- {
		if err := instances[index].Unload(ctx); err != nil {
			joined = errors.Join(joined, fmt.Errorf("plugin %s: %w",
				instances[index].manifest.ID, err))
		}
	}
	return joined
}

func (r *Runtime) snapshot() []*Instance {
	r.mu.Lock()
	defer r.mu.Unlock()
	return orderedInstances(r.instances)
}

func orderedInstances(instances map[string]*Instance) []*Instance {
	ids := make([]string, 0, len(instances))
	for id := range instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ordered := make([]*Instance, 0, len(ids))
	for _, id := range ids {
		ordered = append(ordered, instances[id])
	}
	return ordered
}
