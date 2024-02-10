// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
)

// NewTestLogger returns a new test logger that writes to a buffer
// with slog.JSONHandler. The buffer can be accessed via the Output method.
func NewTestLogger(lvl Level) *TestLogger {
	l := &DefaultLogger{
		lvl: new(slog.LevelVar),
		ctx: context.Background(),
	}
	out := new(bytes.Buffer)

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
	mu  sync.RWMutex
	log *DefaultLogger
	out *bytes.Buffer
}

// Output returns String returns the contents of the unread portion
// of the log output as a string.
func (tl *TestLogger) Output() string {
	tl.mu.RLock()
	defer tl.mu.RUnlock()
	return tl.out.String()
}

func (l *TestLogger) SystemDebug(msg string, attrs ...slog.Attr) {
	l.log.SystemDebug(msg, attrs...)
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

func (l *TestLogger) BUG(msg string, attrs ...slog.Attr) {
	l.log.BUG(msg, attrs...)
}

func (l *TestLogger) Println(msg string, attrs ...slog.Attr) {
	l.log.Println(msg, attrs...)
}

func (l *TestLogger) Printf(format string, v ...any) {
	l.log.Printf(format, v...)
}

func (l *TestLogger) HTTP(method, path string, status int, attrs ...slog.Attr) {
	l.log.HTTP(method, path, status, attrs...)
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
