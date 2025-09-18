// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"log/slog"
	"os"
)

// Logger is the primary interface for logging in the Happy SDK, fully compatible with slog.Logger.
// It wraps slog.Logger to provide standard logging methods (e.g., Debug, Info, Error) and manages
// custom adapters for output handling. Users can treat it as a slog.Logger, with all inherited methods
// delegating to configured adapters. The only exception is that Dispose must be called before the
// application exits or when the logger is no longer needed to release adapter resources and close writers.
type Logger struct {
	*slog.Logger          // Embedded slog.Logger for standard logging methods.
	handler      *handler // Internal handler managing adapters.
}

// New creates a Logger with the given configuration and adapters.
// If no adapters are provided, a default text adapter writing to os.Stdout is used.
// The config is sealed to finalize settings (e.g., level, timestamp format), and the logger is set
// as the default slog.Logger if configured. All adapters are initialized and ready after creation.
func New(config Config, adapters ...Adapter) *Logger {
	configPtr := &config
	configPtr.seal()

	var configuredAdapters []*adapter
	for _, a := range adapters {
		if a == nil {
			continue
		}
		switch aa := a.(type) {
		case ComposableAdapter:
			composed := aa.Compose(*configPtr)
			if composed == nil {
				continue
			}
			if a, ok := composed.(*adapter); ok {
				configuredAdapters = append(configuredAdapters, a)
			}
		case *adapter:
			configuredAdapters = append(configuredAdapters, aa)
		default:
			configuredAdapters = append(configuredAdapters, newAdapter(nil, a))
		}
	}
	if len(configuredAdapters) == 0 {
		if len(adapters) > 0 {
			h := &handler{}
			h.disposed.Store(true)
			return &Logger{
				Logger:  slog.New(DiscardAdapter),
				handler: h,
			}
		} else {
			configuredAdapters = []*adapter{NewTextAdapter(os.Stdout).Compose(config).(*adapter)}
		}
	}

	h := newHandler(*configPtr, configuredAdapters)
	l := &Logger{
		Logger:  slog.New(h),
		handler: h,
	}
	if configPtr.SetSlogOutput {
		slog.SetDefault(l.Logger)
	}

	h.Ready()
	return l
}

// Dispose closes all adapters, releasing their resources and closing writers.
// It must be called before the application exits or when the logger is no longer needed.
// Idempotent, safe for concurrent use.
func (l *Logger) Dispose() error {
	if err := l.handler.Dispose(); err != nil {
		return err
	}
	l.Logger = slog.New(DiscardAdapter)
	return nil
}

// SetLevel updates the logger's minimum log level dynamically,
// affecting all adapters.
func (l *Logger) SetLevel(level Level) {
	if l.handler.disposed.Load() {
		return
	}
	l.handler.level.Set(level.Level())
}

// Flush writes buffered log entries to their writers and syncs data.
// Only affects adapters implementing FlushableAdapter. Safe for concurrent use.
func (l *Logger) Flush() error {
	if l.handler.disposed.Load() {
		return ErrLoggerDisposed
	}
	return l.handler.Flush()
}
