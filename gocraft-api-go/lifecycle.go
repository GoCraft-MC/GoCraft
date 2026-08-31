package gocraft

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

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

func (s *runtimeState) load(pluginID, dataDirectory string) ([]string, error) {
	if s.context != nil {
		return nil, fmt.Errorf("gocraft: plugin is already loaded")
	}
	if pluginID != s.metadata.ID {
		return nil, fmt.Errorf("gocraft: executable is %s, bundle requested %s", s.metadata.ID, pluginID)
	}
	logger := slog.Default().With("plugin", s.metadata.ID)
	s.context = newContext(s.metadata, dataDirectory, logger)
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
