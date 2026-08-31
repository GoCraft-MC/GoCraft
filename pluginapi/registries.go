package pluginapi

import "log/slog"

// Events owns listeners registered by one plugin.
type Events struct{ logger *slog.Logger }

func newEvents(logger *slog.Logger) *Events { return &Events{logger: logger} }

// Commands owns command callbacks registered by one plugin.
type Commands struct{ logger *slog.Logger }

func newCommands(logger *slog.Logger) *Commands { return &Commands{logger: logger} }

// Scheduler owns asynchronous tasks registered by one plugin.
type Scheduler struct{ logger *slog.Logger }

func newScheduler(logger *slog.Logger) *Scheduler { return &Scheduler{logger: logger} }
