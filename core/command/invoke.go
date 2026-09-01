package command

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrUnknownExecutor = errors.New("command executor is not registered")
	ErrPermission      = errors.New("command permission denied")
)

// Execute resolves one typed line against what this sender can see and runs it.
//
// The boolean is the whole point of the signature: false means the line names
// nothing here, so a caller that also owns other commands can fall through to
// them instead of answering "unknown command" on behalf of a registry that was
// never asked. True with an error is the other case — this registry owns the
// line and something about it was wrong, and the error is a sentence to show
// whoever typed it.
//
// Resolution runs against a snapshot, so a branch guarded by a permission the
// sender lacks is invisible rather than refused. Invoke checks permissions
// again on the executor it reached: the snapshot decides what can be seen, the
// registry decides what can be run, and neither trusts the other.
func (r *Registry) Execute(ctx context.Context, sender Sender, line string, resolvers Resolvers) (bool, error) {
	executor, args, err := r.Snapshot(sender).Resolve(line, resolvers)
	switch {
	case errors.Is(err, ErrNoSuchCommand):
		return false, nil
	case err != nil:
		return true, err
	}
	return true, r.invoke(ctx, executor, sender, args, tokensAfterName(line))
}

func (r *Registry) Invoke(ctx context.Context, executor ExecID, sender Sender, args Values) error {
	return r.invoke(ctx, executor, sender, args, nil)
}

func (r *Registry) invoke(ctx context.Context, executor ExecID, sender Sender, args Values, raw []string) error {
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
	return registered.handler(ctx, &Context{Sender: sender, Args: args, Node: executor, Raw: raw})
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

// tokensAfterName is the line as an unmigrated handler expects to read it: the
// words after the command name, with the leading slash gone.
func tokensAfterName(line string) []string {
	tokens := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "/"))
	if len(tokens) == 0 {
		return nil
	}
	return tokens[1:]
}
