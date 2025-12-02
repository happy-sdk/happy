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

	records := queue.Records()

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

// defaultCallerDepth is the stack depth used by convenience methods so that
// the reported source location points to the caller of the helper, not the
// helper itself.
const defaultCallerDepth = 2

// logAt is a small helper that logs at the given level with proper depth and arguments.
// It converts args to slog.Attr using the same logic as slog.Logger methods.
func (l *Logger) logAt(depth int, lvl Level, msg string, args ...any) {
	if l == nil {
		return
	}
	if !l.Enabled(context.Background(), slog.Level(lvl)) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(depth+2, pcs[:])
	r := slog.NewRecord(l.ts(), slog.Level(lvl), msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(context.Background(), r)
}

// logAtContext is a small helper that logs at the given level with context, proper depth and arguments.
func (l *Logger) logAtContext(ctx context.Context, depth int, lvl Level, msg string, args ...any) {
	if l == nil {
		return
	}
	if !l.Enabled(ctx, slog.Level(lvl)) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(depth+2, pcs[:])
	r := slog.NewRecord(l.ts(), slog.Level(lvl), msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// Happy logs a message at the Happy SDK "happy" level.
func (l *Logger) Happy(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelHappy, msg, args...)
}

// HappyContext logs a message at the Happy SDK "happy" level with context.
func (l *Logger) HappyContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelHappy, msg, args...)
}

// DebugPkg logs a message at the package-debug level.
func (l *Logger) DebugPkg(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelDebugPkg, msg, args...)
}

// DebugPkgContext logs a message at the package-debug level with context.
func (l *Logger) DebugPkgContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelDebugPkg, msg, args...)
}

// DebugAddon logs a message at the addon-debug level.
func (l *Logger) DebugAddon(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelDebugAddon, msg, args...)
}

// DebugAddonContext logs a message at the addon-debug level with context.
func (l *Logger) DebugAddonContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelDebugAddon, msg, args...)
}

// Trace logs a message at the trace level.
func (l *Logger) Trace(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelTrace, msg, args...)
}

// TraceContext logs a message at the trace level with context.
func (l *Logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelTrace, msg, args...)
}

// Notice logs a message at the notice level.
func (l *Logger) Notice(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelNotice, msg, args...)
}

// NoticeContext logs a message at the notice level with context.
func (l *Logger) NoticeContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelNotice, msg, args...)
}

// Success logs a message at the success level.
func (l *Logger) Success(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelSuccess, msg, args...)
}

// SuccessContext logs a message at the success level with context.
func (l *Logger) SuccessContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelSuccess, msg, args...)
}

// NotImpl logs a message at the NotImpl level, for unimplemented features.
func (l *Logger) NotImpl(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelNotImpl, msg, args...)
}

// NotImplContext logs a message at the NotImpl level with context.
func (l *Logger) NotImplContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelNotImpl, msg, args...)
}

// NotImplemented is an alias for NotImpl.
func (l *Logger) NotImplemented(msg string, args ...any) {
	l.NotImpl(msg, args...)
}

// NotImplementedContext is an alias for NotImplContext.
func (l *Logger) NotImplementedContext(ctx context.Context, msg string, args ...any) {
	l.NotImplContext(ctx, msg, args...)
}

// Deprecated logs a message at the Deprecated level.
func (l *Logger) Deprecated(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelDepr, msg, args...)
}

// DeprecatedContext logs a message at the Deprecated level with context.
func (l *Logger) DeprecatedContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelDepr, msg, args...)
}

// Out logs a message at the Out level, intended for stdout/stderr style output.
func (l *Logger) Out(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelOut, msg, args...)
}

// OutContext logs a message at the Out level with context.
func (l *Logger) OutContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelOut, msg, args...)
}

// BUG logs a message at the BUG level, intended for critical bugs.
func (l *Logger) BUG(msg string, args ...any) {
	l.logAt(defaultCallerDepth, LevelBUG, msg, args...)
}

// BUGContext logs a message at the BUG level with context.
func (l *Logger) BUGContext(ctx context.Context, msg string, args ...any) {
	l.logAtContext(ctx, defaultCallerDepth, LevelBUG, msg, args...)
}
