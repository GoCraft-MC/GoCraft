package gocraft

import (
	"fmt"
	"runtime/debug"
)

// Register binds a callback to one path through the bundle's command tree:
//
//	ctx.Commands().Register("shop sell <price>", func(call *gocraft.CommandContext) error {
//	    return sell(call.Sender, call.Args.Decimal("price"))
//	})
//
// The path, not the executor id the tree assigns. Ids are chosen by whatever
// built the tree, so naming one here would write it down a second time — free
// to disagree with the first the day a command is inserted above it. A path
// that names nothing is refused at registration, with the paths the bundle does
// declare, rather than becoming a handler that silently never runs.
func (c *Commands) Register(commandPath string, handler CommandHandler) error {
	if handler == nil {
		return fmt.Errorf("gocraft: a command handler is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return fmt.Errorf("gocraft: command registry is disabled")
	}
	executor, err := c.tree.lookup(commandPath)
	if err != nil {
		return err
	}
	if _, duplicate := c.handlers[executor]; duplicate {
		return fmt.Errorf("gocraft: command %q is already registered", commandPath)
	}
	c.handlers[executor] = handler
	return nil
}

func (c *Commands) invoke(executor uint32, call *CommandContext) (replies []string, err error) {
	c.mu.RLock()
	handler, ok := c.handlers[executor]
	active := c.active
	c.mu.RUnlock()
	if !active {
		return nil, fmt.Errorf("gocraft: command registry is disabled")
	}
	if !ok {
		return nil, fmt.Errorf("gocraft: command executor %d is not registered", executor)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			c.logger.Error("plugin command panicked", "executor", executor,
				"panic", recovered, "stack", string(debug.Stack()))
			err = fmt.Errorf("plugin command %d panicked", executor)
		}
	}()
	err = handler(call)
	return append([]string(nil), call.replies...), err
}

func (c *Commands) clear() {
	c.mu.Lock()
	c.active = false
	c.handlers = make(map[uint32]CommandHandler)
	c.mu.Unlock()
}
