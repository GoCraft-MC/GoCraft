package gocraft

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

// loadRequest is what the host sends with LOAD, in the plugin's own words.
type loadRequest struct {
	pluginID      string
	bundlePath    string
	dataDirectory string
	// commandTree is where the tree sits inside the bundle, empty when the
	// plugin declares no commands.
	commandTree string
}

type runtimeState struct {
	metadata       Metadata
	implementation Plugin
	context        *pluginContext
	initialized    bool
	enabled        bool
}

func newRuntimeState(metadata Metadata, implementation Plugin) *runtimeState {
	return &runtimeState{metadata: metadata, implementation: implementation}
}

func (s *runtimeState) load(request loadRequest) ([]string, error) {
	if s.context != nil {
		return nil, fmt.Errorf("gocraft: plugin is already loaded")
	}
	if request.pluginID != s.metadata.ID {
		return nil, fmt.Errorf("gocraft: executable is %s, bundle requested %s", s.metadata.ID, request.pluginID)
	}
	// Read before anything is constructed: a bundle whose command tree cannot
	// be read is a bundle whose commands would never run, and failing here
	// gives the admin a reason instead of a plugin that loads and does half of
	// what it says.
	tree, err := loadCommandTree(request.bundlePath, request.commandTree)
	if err != nil {
		return nil, err
	}
	logger := slog.Default().With("plugin", s.metadata.ID)
	s.context = newContext(s.metadata, request.dataDirectory, tree, logger)
	if err := s.call("load", func() error { return s.implementation.OnLoad(s.context) }); err != nil {
		s.cleanup()
		s.context = nil
		return nil, err
	}
	s.initialized = true
	if err := s.call("enable", s.implementation.OnEnable); err != nil {
		disableErr := s.disable()
		if disableErr != nil {
			logger.Error("plugin cleanup failed", "err", disableErr)
		}
		return nil, err
	}
	s.enabled = true
	return s.context.events.registeredTypes(), nil
}

func (s *runtimeState) disable() error {
	if s.context == nil {
		return nil
	}
	s.cleanup()
	var err error
	if s.initialized {
		err = s.call("disable", s.implementation.OnDisable)
	}
	s.context = nil
	s.initialized = false
	s.enabled = false
	return err
}

func (s *runtimeState) cleanup() {
	s.context.events.clear()
	s.context.commands.clear()
	s.context.scheduler.stop()
}

func (s *runtimeState) call(phase string, callback func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("plugin lifecycle panicked", "plugin", s.metadata.ID,
				"phase", phase, "panic", recovered, "stack", string(debug.Stack()))
			err = fmt.Errorf("plugin %s %s panicked: %v", s.metadata.ID, phase, recovered)
		}
	}()
	if err := callback(); err != nil {
		return fmt.Errorf("plugin %s %s: %w", s.metadata.ID, phase, err)
	}
	return nil
}
