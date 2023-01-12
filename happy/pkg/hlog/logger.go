// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package hlog

import (
	"context"
	"strings"
	"time"

	"github.com/mkungla/happy/pkg/vars"
	"github.com/mkungla/happy/sdk/sarg"
	"golang.org/x/exp/slog"
)

type Logger struct {
	slog *slog.Logger
}

// Debug logs at LevelDebug.
func (l *Logger) SystemDebug(msg string, args ...any) {
	l.LogDepth(0, LevelSystemDebug, msg, args...)
}

// Debug logs at LevelDebug.
func (l *Logger) Debug(msg string, args ...any) {
	l.LogDepth(0, LevelDebug, msg, args...)
}

// Info logs at LevelInfo.
func (l *Logger) Info(msg string, args ...any) {
	l.LogDepth(0, LevelInfo, msg, args...)
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
	l.LogDepth(0, LevelTask, "", slog.Group("task", args...))
	return t
}

func (l *Logger) Ok(msg string, args ...any) {
	l.LogDepth(0, LevelOk, msg, args...)
}

func (l *Logger) Notice(msg string, args ...any) {
	l.LogDepth(0, LevelNotice, msg, args...)
}

// Warn logs at LevelWarn.
func (l *Logger) Warn(msg string, args ...any) {
	l.LogDepth(0, LevelWarn, msg, args...)
}

// Debug logs at LevelDebug.
func (l *Logger) NotImplemented(msg string, args ...any) {
	l.LogDepth(0, LevelNotImplemented, msg, args...)
}

// Warn logs at LevelWarn.
func (l *Logger) Deprecated(msg string, args ...any) {
	l.LogDepth(0, LevelDeprecated, msg, args...)
}

func (l *Logger) Issue(msg string, args ...any) {
	l.LogDepth(0, LevelIssue, msg, args...)
}

// Error logs at LevelError.
// If err is non-nil, Error appends Any(ErrorKey, err)
// to the list of attributes.
func (l *Logger) Error(msg string, err error, args ...any) {
	// Would need to have workaround for this allocation
	if err != nil {
		errmsgs := strings.Split(err.Error(), "\n")
		for _, emsg := range errmsgs {
			l.LogDepth(0, LevelError, msg, slog.String("err", emsg))
		}
	}
	if len(args) > 0 {
		l.LogDepth(0, LevelError, msg, args...)
	}
}

func (l *Logger) Out(msg string, args ...any) {
	l.LogDepth(0, LevelOut, msg, args...)
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
	l.LogDepth(0, level, msg, args...)
}

// LogDepth is like [Logger.Log], but accepts a call depth to adjust the
// file and line number in the log record. 0 refers to the caller
// of LogDepth; 1 refers to the caller's caller; and so on.
func (l *Logger) LogDepth(calldepth int, level Level, msg string, args ...any) {
	if len(args) > 0 {
		var sargs []any
		for _, arg := range args {
			if v, ok := arg.(vars.Variable); ok {
				var attr slog.Attr
				switch v.Kind() {
				case vars.KindBool:
					attr = slog.Bool(v.Name(), v.Bool())
				case vars.KindFloat32, vars.KindFloat64:
					attr = slog.Float64(v.Name(), v.Float64())
				case vars.KindInt:
					attr = slog.Int(v.Name(), v.Int())
				case vars.KindInt64:
					attr = slog.Int64(v.Name(), v.Int64())
				case vars.KindUint64:
					attr = slog.Uint64(v.Name(), v.Uint64())
				default:
					attr = slog.Any(v.Name(), v.Any())
				}
				sargs = append(sargs, attr)
			} else {
				sargs = append(sargs, arg)
			}
		}
		args = sargs
	}

	l.slog.LogDepth(calldepth+1, slog.Level(level), msg, args...)
}

// LogAttrs is a more efficient version of [Logger.Log] that accepts only Attrs.
func (l *Logger) LogAttrs(level Level, msg string, attrs ...slog.Attr) {
	l.LogAttrsDepth(0, level, msg, attrs...)
}

// LogAttrsDepth is like [Logger.LogAttrs], but accepts a call depth argument
// which it interprets like [Logger.LogDepth].
func (l *Logger) LogAttrsDepth(calldepth int, level Level, msg string, attrs ...slog.Attr) {
	l.slog.LogAttrsDepth(calldepth, slog.Level(level), msg, attrs...)
}
