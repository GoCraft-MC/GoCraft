package pluginapi

import (
	"log/slog"
	"sync"
)

// Events owns listeners registered by one plugin.
type Events struct {
	mu        sync.RWMutex
	logger    *slog.Logger
	listeners map[string][]EventHandler
	active    bool
}

func newEvents(logger *slog.Logger) *Events {
	return &Events{logger: logger, listeners: make(map[string][]EventHandler), active: true}
}

// Commands owns command callbacks registered by one plugin.
type Commands struct{ logger *slog.Logger }

func newCommands(logger *slog.Logger) *Commands { return &Commands{logger: logger} }

// Scheduler owns asynchronous tasks registered by one plugin.
type Scheduler struct{ logger *slog.Logger }

func newScheduler(logger *slog.Logger) *Scheduler { return &Scheduler{logger: logger} }
