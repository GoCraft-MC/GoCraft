package pluginapi

import (
	"fmt"
	"runtime/debug"
)

// Register binds a local executor ID from commands.pb to a callback.
func (c *Commands) Register(executor uint32, handler CommandHandler) error {
	if executor == 0 || handler == nil {
		return fmt.Errorf("pluginapi: command executor and handler are required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.active {
		return fmt.Errorf("pluginapi: command registry is disabled")
	}
	if _, duplicate := c.handlers[executor]; duplicate {
		return fmt.Errorf("pluginapi: command executor %d is already registered", executor)
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
		return nil, fmt.Errorf("pluginapi: command registry is disabled")
	}
	if !ok {
		return nil, fmt.Errorf("pluginapi: command executor %d is not registered", executor)
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
