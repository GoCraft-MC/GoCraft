package command

import (
	"context"
	"errors"
)

var (
	ErrUnknownExecutor = errors.New("command executor is not registered")
	ErrPermission      = errors.New("command permission denied")
)

func (r *Registry) Invoke(ctx context.Context, executor ExecID, sender Sender, args Values) error {
	r.mu.RLock()
	registered, ok := r.handlers[executor]
	key, _ := sourceKey(registered.source)
	entry := r.entries[key]
	allowed := ok && executorAllowed(entry.root.Children, executor, sender, true)
	r.mu.RUnlock()
	if !ok {
		return ErrUnknownExecutor
	}
	if !allowed {
		return ErrPermission
	}
	return registered.handler(ctx, &Context{Sender: sender, Args: args, Node: executor})
}

func executorAllowed(nodes []Node, executor ExecID, sender Sender, parentAllowed bool) bool {
	for _, node := range nodes {
		allowed := parentAllowed
		switch typed := node.(type) {
		case Literal:
			if typed.Permission != "" {
				allowed = allowed && sender != nil && sender.Has(typed.Permission)
			}
			if allowed && typed.Exec == executor {
				return true
			}
			if executorAllowed(typed.Children, executor, sender, allowed) {
				return true
			}
		case Argument:
			if allowed && typed.Exec == executor {
				return true
			}
			if executorAllowed(typed.Children, executor, sender, allowed) {
				return true
			}
		}
	}
	return false
}
