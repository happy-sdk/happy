// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"
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

// LogDepth logs a message with additional context at a given depth.
// The depth is the number of stack frames to ascend when logging the message.
// It is useful only when AddSource is enabled.
func (l *Logger) LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr) error {
	if !l.Enabled(context.Background(), slog.Level(lvl)) {
		return nil
	}
	var pcs [1]uintptr
	runtime.Callers(depth+2, pcs[:])
	r := slog.NewRecord(l.ts(), slog.Level(lvl), msg, pcs[0])
	r.AddAttrs(attrs...)
	return l.Handler().Handle(context.Background(), r)
}

// Consume drains all queued records into the Logger.
// Call *QueueLogger Dispose dont want to use it anymore to release
// intrnal BufferAdapter
// It returns the number of records consumed and any error encountered.
func (l *Logger) Consume(queue *QueueLogger) (int, error) {
	if queue.queue.disposed.Load() {
		return 0, fmt.Errorf("%w: QueueLogger disposed", ErrAdapter)
	}

	if err := queue.adapter.Flush(); err != nil {
		return 0, err
	}

	queue.queue.mu.Lock()
	records := queue.queue.buf
	queue.queue.mu.Unlock()

	processed, err := l.handler.queueHandle(records)
	if err != nil {
		return processed, err
	}
	if err := l.Flush(); err != nil {
		return processed, err
	}
	return processed, nil
}

func (l *Logger) Level() Level {
	return Level(l.handler.level.Level())
}

func (l *Logger) ts() time.Time {
	state := l.handler.state.Load()
	return time.Now().In(state.config.TimeLocation)
}
