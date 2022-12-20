// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happylog

import (
	"io"
	"os"
	"sync/atomic"

	"github.com/mkungla/happy/sdk/sarg"
	"golang.org/x/exp/slog"
)

var defaultLogger atomic.Value

func init() {
	defaultLogger.Store(New(Config{}.NewHandler(os.Stdout)))
}

// Default returns the default Logger.
func Default() *Logger { return defaultLogger.Load().(*Logger) }

// SetDefault makes l the default Logger.
// After this call, output from the log package's default Logger
// (as with [log.Print], etc.) will be logged at LevelInfo using l's Handler.
func SetDefault(l *Logger, stdlog bool) {
	defaultLogger.Store(l)
	if stdlog {
		slog.SetDefault(l.slog)
	}
}

// With calls Logger.With on the default logger.
func With(args ...any) *Logger {
	return Default().With(args...)
}

func WithGroup(name string) *Logger {
	return Default().WithGroup(name)
}

func New(h slog.Handler) *Logger { return &Logger{slog: slog.New(h)} }

// Debug calls Logger.Debug on the default logger.
func Debug(msg string, args ...any) {
	Default().LogDepth(0, LevelDebug, msg, args...)
}

// Debug calls Logger.Debug on the default logger.
func SystemDebug(msg string, args ...any) {
	Default().LogDepth(0, LevelSystemDebug, msg, args...)
}

// Info calls Logger.Info on the default logger.
func Info(msg string, args ...any) {
	Default().LogDepth(0, LevelInfo, msg, args...)
}

func Task(name string, args ...slog.Attr) sarg.Task {
	return Default().Task(name, args...)
}

func Ok(msg string, args ...any) {
	Default().LogDepth(0, LevelOk, msg, args...)
}

func Notice(msg string, args ...any) {
	Default().LogDepth(0, LevelNotice, msg, args...)
}

func NotImplemented(msg string, args ...any) {
	Default().LogDepth(0, LevelNotImplemented, msg, args...)
}

func Deprecated(msg string, args ...any) {
	Default().LogDepth(0, LevelDeprecated, msg, args...)
}

func Issue(msg string, args ...any) {
	Default().LogDepth(0, LevelIssue, msg, args...)
}

// Warn calls Logger.Warn on the default logger.
func Warn(msg string, args ...any) {
	Default().LogDepth(0, LevelWarn, msg, args...)
}

// Error calls Logger.Error on the default logger.
func Error(msg string, err error, args ...any) {
	if err != nil {
		// Would need to have workaround for this allocation
		args = append([]any{slog.Any("err", err)}, args...)
	}
	Default().LogDepth(0, LevelError, msg, args...)
}

func Out(msg string, args ...any) {
	Default().LogDepth(0, LevelOut, msg, args...)
}

// Log calls Logger.Log on the default logger.
func Log(level Level, msg string, args ...any) {
	Default().LogDepth(0, level, msg, args...)
}

// LogAttrs calls Logger.LogAttrs on the default logger.
func LogAttrs(level Level, msg string, attrs ...slog.Attr) {
	Default().LogAttrsDepth(1, level, msg, attrs...)
}

type Config struct {
	Options slog.HandlerOptions
	JSON    bool
	Colors  bool
	Secrets []string
}

// NewJSONHandler creates a JSONHandler that writes to w,
// using the default options.
func (cnf Config) NewHandler(w io.Writer) slog.Handler {
	if cnf.JSON {
		opts := slog.HandlerOptions{
			Level:     cnf.Options.Level,
			AddSource: cnf.Options.AddSource,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if rep := cnf.Options.ReplaceAttr; rep != nil {
					a = cnf.Options.ReplaceAttr(groups, a)
				}

				for _, secret := range cnf.Secrets {
					if a.Key == secret {
						a.Value = slog.StringValue("*****")
					}
				}
				if a.Key == slog.LevelKey {
					lvl, ok := a.Value.Any().(slog.Level)
					if ok {
						return slog.String("level", Level(lvl).String())
					}
				}
				return a
			},
		}
		return opts.NewJSONHandler(w)

	}
	h := &Handler{
		w:      w,
		colors: cnf.Colors,
	}
	h.opts = slog.HandlerOptions{
		Level:     cnf.Options.Level,
		AddSource: cnf.Options.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if rep := cnf.Options.ReplaceAttr; rep != nil {
				a = cnf.Options.ReplaceAttr(groups, a)
			}

			for _, secret := range cnf.Secrets {
				if a.Key == secret {
					a.Value = slog.StringValue("*****")
				}
			}
			return a
		},
	}

	return h
}

const badKey = "!BADKEY"

// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}
		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}
