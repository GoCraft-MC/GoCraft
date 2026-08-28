package plugin

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

const defaultEventBudget = 2 * time.Millisecond

type subscriber struct {
	id       string
	priority Priority
	instance Instance
	health   *healthTracker
}

// Bus routes events to subscriptions declared before plugin code starts.
type Bus struct {
	ctx    context.Context
	budget time.Duration

	mu     sync.RWMutex
	subs   map[string][]*subscriber
	health map[string]*healthTracker
}

func NewBus(ctx context.Context, budget time.Duration) *Bus {
	if ctx == nil {
		ctx = context.Background()
	}
	if budget <= 0 {
		budget = defaultEventBudget
	}
	return &Bus{ctx: ctx, budget: budget, subs: make(map[string][]*subscriber), health: make(map[string]*healthTracker)}
}

func (b *Bus) Attach(instance Instance) error {
	manifest := instance.Manifest()
	if manifest.ID == "" {
		return fmt.Errorf("plugin: empty manifest id")
	}
	seen := make(map[string]struct{}, len(manifest.Subscriptions))
	for _, declared := range manifest.Subscriptions {
		if declared.Event == "" {
			return fmt.Errorf("plugin %s: empty event subscription", manifest.ID)
		}
		if _, duplicate := seen[declared.Event]; duplicate {
			return fmt.Errorf("plugin %s: duplicate subscription to %s", manifest.ID, declared.Event)
		}
		seen[declared.Event] = struct{}{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for event := range seen {
		for _, existing := range b.subs[event] {
			if existing.id == manifest.ID {
				return fmt.Errorf("plugin %s: duplicate subscription to %s", manifest.ID, event)
			}
		}
	}
	tracker := b.health[manifest.ID]
	if tracker == nil {
		tracker = newHealthTracker()
		b.health[manifest.ID] = tracker
	}
	for _, declared := range manifest.Subscriptions {
		sub := &subscriber{id: manifest.ID, priority: declared.Priority, instance: instance, health: tracker}
		b.subs[declared.Event] = append(b.subs[declared.Event], sub)
		sort.Slice(b.subs[declared.Event], func(i, j int) bool {
			left, right := b.subs[declared.Event][i], b.subs[declared.Event][j]
			if left.priority != right.priority {
				return priorityRank(left.priority) < priorityRank(right.priority)
			}
			return left.id < right.id
		})
	}
	return nil
}

func (b *Bus) Detach(pluginID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.health, pluginID)
	for event, subscribers := range b.subs {
		kept := subscribers[:0]
		for _, sub := range subscribers {
			if sub.id != pluginID {
				kept = append(kept, sub)
			}
		}
		if len(kept) == 0 {
			delete(b.subs, event)
		} else {
			b.subs[event] = kept
		}
	}
}

// Health reports event failures and starvation for a loaded plugin.
func (b *Bus) Health(pluginID string) (HealthSnapshot, bool) {
	b.mu.RLock()
	tracker, ok := b.health[pluginID]
	b.mu.RUnlock()
	if !ok {
		return HealthSnapshot{}, false
	}
	return tracker.snapshot(time.Now()), true
}

func priorityRank(priority Priority) int {
	if priority == PriorityMonitor {
		return 5
	}
	return int(PriorityHighest - priority)
}
