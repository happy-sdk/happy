// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type DefaultLogger struct {
	mu      sync.RWMutex
	adapter Adapter

	ctx   context.Context
	lvl   *slog.LevelVar
	tsloc *time.Location
	log   *slog.Logger
}

func New(a Adapter) *DefaultLogger {
	if a == nil {
		a = NewTextAdapter(context.Background(), os.Stdout, nil)
	}
	opts := a.Options()
	if opts == nil {
		opts = DefaultOptions()
	}

	var tsloc *time.Location
	if opts.TimeLocation != nil {
		tsloc = opts.TimeLocation
	} else {
		tsloc = time.Local
	}

	ctx := context.WithoutCancel(a.Context())

	lvlvar := opts.LevelVar
	if lvlvar == nil {
		lvlvar = new(slog.LevelVar)
	}

	lvlvar.Set(slog.Level(opts.Level))

	logger := &DefaultLogger{
		adapter: a,
		ctx:     ctx,
		lvl:     lvlvar,
		tsloc:   tsloc,
		log:     slog.New(a.Handler()),
	}

	if opts.SetSlogOutput {
		slog.SetDefault(logger.log)
	}
	return logger
}

func (l *DefaultLogger) Debug(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlDebug, msg, attrs...)
}

func (l *DefaultLogger) Info(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlInfo, msg, attrs...)
}

func (l *DefaultLogger) Ok(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlOk, msg, attrs...)
}

func (l *DefaultLogger) Notice(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlNotice, msg, attrs...)
}

func (l *DefaultLogger) Warn(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlWarn, msg, attrs...)
}

func (l *DefaultLogger) NotImplemented(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlNotImplemented, msg, attrs...)
}

func (l *DefaultLogger) Deprecated(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlDeprecated, msg, attrs...)
}

func (l *DefaultLogger) Error(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlError, msg, attrs...)
}

func (l *DefaultLogger) Errors(err error, attrs ...slog.Attr) {
	var errs []error
	// Use errors.Unwrap to iterate through wrapped errors.
	for e := err; e != nil; {
		// Check if e is a joined error or a single error.
		if unwrapped, ok := e.(interface{ Unwrap() []error }); ok {
			// If it supports Unwrap() []error, append all errors.
			errs = append(errs, unwrapped.Unwrap()...)
			break
		}
		// Try unwrapping single error.
		if next := errors.Unwrap(e); next != nil {
			errs = append(errs, e)
			e = next
		} else {
			errs = append(errs, e)
			break
		}
	}
	for _, err := range errs {
		l.logDepth(lvlError, err.Error(), attrs...)
	}
}

func (l *DefaultLogger) BUG(msg string, attrs ...slog.Attr) {
	l.logDepth(lvlBUG, msg, attrs...)
}

func (l *DefaultLogger) Println(line string, attrs ...slog.Attr) {
	line = strings.TrimRight(line, "\n")
	if !testing.Testing() {
		line += "\n"
	}
	l.logDepth(lvlAlways, line, attrs...)
}

func (l *DefaultLogger) Printf(format string, v ...any) {
	l.logDepth(lvlAlways, fmt.Sprintf(format, v...))
}

func (l *DefaultLogger) HTTP(status int, method, path string, attrs ...slog.Attr) {

	if status < 400 && l.log.Enabled(l.ctx, lvlInfo) {
		l.http(status, method, path, attrs...)
	} else if status < 500 && l.log.Enabled(l.ctx, lvlWarn) {
		l.http(status, method, path, attrs...)
	} else if status < 600 && l.log.Enabled(l.ctx, lvlError) {
		l.http(status, method, path, attrs...)
	} else {
		if l.log.Enabled(l.ctx, lvlError) {
			attrs = append(attrs, slog.String("err", "invalid status code"))
			l.http(status, method, path, attrs...)
		}
	}
}

func (l *DefaultLogger) Enabled(lvl Level) bool { return l.log.Enabled(l.ctx, slog.Level(lvl)) }

func (l *DefaultLogger) Level() Level { return Level(l.lvl.Level()) }

func (l *DefaultLogger) SetLevel(lvl Level) {
	l.lvl.Set(slog.Level(lvl))
}

// LogDepth logs a message with additional context at a given depth.
// The depth is the number of stack frames to ascend when logging the message.
// It is useful only when AddSource is enabled.
func (l *DefaultLogger) LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr) {
	if !l.log.Enabled(l.ctx, slog.Level(lvl)) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(depth+2, pcs[:])
	r := slog.NewRecord(l.ts(), slog.Level(lvl), msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = l.log.Handler().Handle(l.ctx, r)
}

func (l *DefaultLogger) Handle(r slog.Record) error {
	if !l.log.Enabled(l.ctx, r.Level) {
		return nil
	}
	return l.log.Handler().Handle(l.ctx, r)
}

func (l *DefaultLogger) Logger() *slog.Logger {
	return l.log
}

func (l *DefaultLogger) ConsumeQueue(queue *QueueLogger) error {
	records := queue.Consume()
	for _, r := range records {
		if err := l.Handle(r.Record(l.tsloc)); err != nil {
			return err
		}
	}
	return nil
}

func (l *DefaultLogger) AttachAdapter(adapter Adapter) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if adapter == nil {
		return fmt.Errorf("%w: AttachAdapter got nil adapter", Error)
	}

	a := NewCombinedAdapter(l.adapter, adapter)
	l.adapter = a
	l.log = slog.New(a.Handler())

	if adapter.Options().SetSlogOutput {
		slog.SetDefault(l.log)
	}

	return nil
}

func (l *DefaultLogger) SetAdapter(adapter Adapter) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if adapter == nil {
		return fmt.Errorf("%w: SetAdapter got nil adapter", Error)
	}

	l.adapter = adapter
	l.log = slog.New(adapter.Handler())

	if adapter.Options().SetSlogOutput {
		slog.SetDefault(l.log)
	}

	return nil
}

func (l *DefaultLogger) Options() (*Options, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.adapter.Options(), nil
}

func (l *DefaultLogger) Dispose() error {
	if l.adapter != nil {
		return l.adapter.Dispose()
	}
	return nil
}

func (l *DefaultLogger) logDepth(lvl slog.Level, msg string, attrs ...slog.Attr) {
	if !l.log.Enabled(l.ctx, lvl) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(l.ts(), lvl, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = l.log.Handler().Handle(l.ctx, r)
}

func (l *DefaultLogger) http(status int, method, path string, attrs ...slog.Attr) {
	if ch, ok := l.log.Handler().(httpHandler); ok {
		ch.http(status, method, path, attrs...)
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(l.ts(), lvlAlways, fmt.Sprintf("[%-8s %-3s] %s", method, fmt.Sprint(status), path), pcs[0])
	r.AddAttrs(attrs...)
	_ = l.log.Handler().Handle(l.ctx, r)
}

func (l *DefaultLogger) ts() time.Time {
	if l.tsloc == nil {
		panic("logging: time location is nil")
	}
	return time.Now().In(l.tsloc)
}
