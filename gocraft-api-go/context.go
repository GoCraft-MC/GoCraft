package gocraft

import "log/slog"

type pluginContext struct {
	metadata  Metadata
	data      string
	logger    *slog.Logger
	events    *Events
	commands  *Commands
	scheduler *Scheduler
}

func newContext(metadata Metadata, data string, tree *commandTree, logger *slog.Logger) *pluginContext {
	return &pluginContext{
		metadata:  metadata,
		data:      data,
		logger:    logger,
		events:    newEvents(logger),
		commands:  newCommands(logger, tree),
		scheduler: newScheduler(logger),
	}
}

func (c *pluginContext) Metadata() Metadata    { return c.metadata }
func (c *pluginContext) DataDirectory() string { return c.data }
func (c *pluginContext) Logger() *slog.Logger  { return c.logger }
func (c *pluginContext) Events() *Events       { return c.events }
func (c *pluginContext) Commands() *Commands   { return c.commands }
func (c *pluginContext) Scheduler() *Scheduler { return c.scheduler }
