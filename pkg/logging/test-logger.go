// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// safeBuffer wraps bytes.Buffer with mutex protection
type safeBuffer struct {
	mu  sync.RWMutex
	buf bytes.Buffer
}

func (tsb *safeBuffer) Write(p []byte) (n int, err error) {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	return tsb.buf.Write(p)
}

func (tsb *safeBuffer) String() string {
	tsb.mu.RLock()
	defer tsb.mu.RUnlock()
	return tsb.buf.String()
}

// NewTestLogger returns a new test logger that writes to a buffer
// with slog.JSONHandler. The buffer can be accessed via the Output method.
func NewTestLogger(lvl Level) *TestLogger {
	l := &DefaultLogger{
		lvl:   new(slog.LevelVar),
		ctx:   context.Background(),
		tsloc: time.Local,
	}
	out := &safeBuffer{}
	l.lvl.Set(slog.Level(lvl))
	h := slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: l.lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				a.Value = slog.StringValue(Level(level).String())
			}
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
		AddSource: false,
	})
	l.log = slog.New(h)
	return &TestLogger{log: l, out: out}
}

type TestLogger struct {
	log *DefaultLogger
	out *safeBuffer
}

// Output returns String returns the contents of the unread portion
// of the log output as a string.
func (tl *TestLogger) Output() string {
	return tl.out.String()
}

func (l *TestLogger) Debug(msg string, attrs ...slog.Attr) {
	l.log.Debug(msg, attrs...)
}

func (l *TestLogger) Info(msg string, attrs ...slog.Attr) {
	l.log.Info(msg, attrs...)
}

func (l *TestLogger) Ok(msg string, attrs ...slog.Attr) {
	l.log.Ok(msg, attrs...)
}

func (l *TestLogger) Notice(msg string, attrs ...slog.Attr) {
	l.log.Notice(msg, attrs...)
}

func (l *TestLogger) Warn(msg string, attrs ...slog.Attr) {
	l.log.Warn(msg, attrs...)
}

func (l *TestLogger) NotImplemented(msg string, attrs ...slog.Attr) {
	l.log.NotImplemented(msg, attrs...)
}

func (l *TestLogger) Deprecated(msg string, attrs ...slog.Attr) {
	l.log.Deprecated(msg, attrs...)
}

func (l *TestLogger) Error(msg string, attrs ...slog.Attr) {
	l.log.Error(msg, attrs...)
}

func (l *TestLogger) Errors(err error, attrs ...slog.Attr) {
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
		l.log.Error(err.Error(), attrs...)
	}
}

func (l *TestLogger) BUG(msg string, attrs ...slog.Attr) {
	l.log.BUG(msg, attrs...)
}

func (l *TestLogger) Println(msg string, attrs ...slog.Attr) {
	l.log.Println(msg, attrs...)
}

func (l *TestLogger) Printf(format string, v ...any) {
	l.log.Printf(format, v...)
}

func (l *TestLogger) HTTP(status int, method, path string, attrs ...slog.Attr) {
	l.log.HTTP(status, method, path, attrs...)
}

func (l *TestLogger) Enabled(lvl Level) bool { return l.log.Enabled(lvl) }
func (l *TestLogger) Level() Level           { return l.log.Level() }
func (l *TestLogger) SetLevel(lvl Level)     { l.log.SetLevel(lvl) }

func (l *TestLogger) LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr) {
	l.log.LogDepth(depth, lvl, msg, attrs...)
}

func (l *TestLogger) Handle(r slog.Record) error {
	return l.log.Handle(r)
}

func (l *TestLogger) Logger() *slog.Logger {
	return l.log.log
}

func (l *TestLogger) ConsumeQueue(queue *QueueLogger) error {
	records := queue.Consume()
	for _, r := range records {
		if err := l.Handle(r.Record(l.log.tsloc)); err != nil {
			return err
		}
	}
	return nil
}

func (l *TestLogger) Dispose() error {
	if l.log != nil {
		return l.log.Dispose()
	}
	return nil
}

func (l *TestLogger) AttachAdapter(adapter Adapter) error {
	return fmt.Errorf("%w: can not attach adapter to TestLogger", Error)
}

func (l *TestLogger) SetAdapter(adapter Adapter) error {
	return fmt.Errorf("%w: can not set adapter to TestLogger", Error)
}

func (l *TestLogger) Options() (*Options, error) {
	return nil, fmt.Errorf("%w: TestLogger does not return options", Error)
}
