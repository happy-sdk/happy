// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happylog

import (
	"context"
	"time"

	"github.com/mkungla/happy/sdk/sarg"
	"golang.org/x/exp/slog"
)

type Logger struct {
	slog *slog.Logger
}

// Debug logs at LevelDebug.
func (l *Logger) SystemDebug(msg string, args ...any) {
	l.slog.LogDepth(0, levelSystemDebug, msg, args...)
}

// Debug logs at LevelDebug.
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.LogDepth(0, levelDebug, msg, args...)
}

// Info logs at LevelInfo.
func (l *Logger) Info(msg string, args ...any) {
	l.slog.LogDepth(0, levelInfo, msg, args...)
}

func (l *Logger) Task(name string, args ...slog.Attr) sarg.Task {
	t := sarg.Task{
		Name:    name,
		Started: time.Now(),
		Args:    args,
	}

	args = append([]slog.Attr{
		slog.String("name", name),
	}, args...)
	l.slog.LogDepth(0, levelTask, "", slog.Group("task", args...))
	return t
}

func (l *Logger) Ok(msg string, args ...any) {
	l.slog.LogDepth(0, levelOk, msg, args...)
}

func (l *Logger) Notice(msg string, args ...any) {
	l.slog.LogDepth(0, levelNotice, msg, args...)
}

// Warn logs at LevelWarn.
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.LogDepth(0, levelWarn, msg, args...)
}

// Debug logs at LevelDebug.
func (l *Logger) NotImplemented(msg string, args ...any) {
	l.slog.LogDepth(0, levelNotImplemented, msg, args...)
}

// Warn logs at LevelWarn.
func (l *Logger) Deprecated(msg string, args ...any) {
	l.slog.LogDepth(0, levelDeprecated, msg, args...)
}

func (l *Logger) Issue(msg string, args ...any) {
	l.slog.LogDepth(0, levelIssue, msg, args...)
}

// Error logs at LevelError.
// If err is non-nil, Error appends Any(ErrorKey, err)
// to the list of attributes.
func (l *Logger) Error(msg string, err error, args ...any) {
	if err != nil {
		// Would need to have workaround for this allocation
		args = append([]any{slog.Any("err", err)}, args...)
	}
	l.slog.LogDepth(0, levelError, msg, args...)
}

func (l *Logger) Out(msg string, args ...any) {
	l.slog.LogDepth(0, levelOut, msg, args...)
}

// Handler returns l's Handler.
func (l *Logger) Handler() slog.Handler { return l.slog.Handler() }

// Context returns l's context.
func (l *Logger) Context() context.Context { return l.slog.Context() }

// With returns a new Logger that includes the given arguments, converted to
// Attrs as in [Logger.Log]. The Attrs will be added to each output from the
// Logger.
//
// The new Logger's handler is the result of calling WithAttrs on the receiver's
// handler.
func (l *Logger) With(args ...any) *Logger {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return New(l.slog.Handler().WithAttrs(attrs))
}

// WithGroup returns a new Logger that starts a group. The keys of all
// attributes added to the Logger will be qualified by the given name.
//
// The new Logger's handler is the result of calling WithGroup on the receiver's
// handler.
func (l *Logger) WithGroup(name string) *Logger {
	return New(l.Handler().WithGroup(name))
}

// WithContext returns a new Logger with the same handler
// as the receiver and the given context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	l2 := *l
	l2.slog = l2.slog.WithContext(ctx)
	return &l2
}

// Enabled reports whether l emits log records at the given level.
func (l *Logger) Enabled(level Level) bool {
	return l.Handler().Enabled(slog.Level(level))
}

// Log emits a log record with the current time and the given level and message.
// The Record's Attrs consist of the Logger's attributes followed by
// the Attrs specified by args.
//
// The attribute arguments are processed as follows:
//   - If an argument is an Attr, it is used as is.
//   - If an argument is a string and this is not the last argument,
//     the following argument is treated as the value and the two are combined
//     into an Attr.
//   - Otherwise, the argument is treated as a value with key "!BADKEY".
func (l *Logger) Log(level Level, msg string, args ...any) {
	l.slog.LogDepth(0, slog.Level(level), msg, args...)
}

// LogDepth is like [Logger.Log], but accepts a call depth to adjust the
// file and line number in the log record. 0 refers to the caller
// of LogDepth; 1 refers to the caller's caller; and so on.
func (l *Logger) LogDepth(calldepth int, level Level, msg string, args ...any) {
	l.slog.LogDepth(calldepth+1, slog.Level(level), msg, args...)
}

// LogAttrs is a more efficient version of [Logger.Log] that accepts only Attrs.
func (l *Logger) LogAttrs(level Level, msg string, attrs ...slog.Attr) {
	l.slog.LogAttrsDepth(0, slog.Level(level), msg, attrs...)
}

// LogAttrsDepth is like [Logger.LogAttrs], but accepts a call depth argument
// which it interprets like [Logger.LogDepth].
func (l *Logger) LogAttrsDepth(calldepth int, level Level, msg string, attrs ...slog.Attr) {
	l.slog.LogAttrsDepth(calldepth, slog.Level(level), msg, attrs...)
}
