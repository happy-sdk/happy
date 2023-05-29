// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

type Logger struct {
	ctx          context.Context
	level        *levelVar
	replaceFuncs []AttrReplaceFunc
	flags        RecordFlag

	// timestamps
	tloc     *time.Location
	tlay     string
	handlers []Handler
}

// New creates and returns new logger.
func New(config Config) (*Logger, error) {
	l := &Logger{
		ctx:   context.Background(),
		level: &levelVar{},
	}
	l.level.Set(config.Level)

	if config.TimeLoc == "" {
		config.TimeLoc = "Local"
	}
	if config.TimeLayout == "" {
		config.TimeLayout = "2006/01/02 15:04:05"
	}
	tloc, err := time.LoadLocation(config.TimeLoc)
	if err != nil {
		return nil, err
	}
	l.tloc = tloc
	l.tlay = config.TimeLayout
	l.replaceFuncs = config.attrReplacers()

	l.replaceFuncs = append(l.replaceFuncs, config.ReplaceAttr...)

	// Add default handler.
	if l.handlers == nil {
		flags := WithRecordTimestamp | WithRecordLevel | WithRecordMessage | WithRecordError | WithRecordData
		if config.AddSource {
			flags |= WithRecordSource
		}

		switch config.DefaultHandler {
		case WithColoredHandler:
			l.handlers = append(l.handlers, newColoredHandler(l.level.Level(), flags, os.Stderr))
		case WithTextHandler:
			l.handlers = append(l.handlers, newTextHandler(l.level.Level(), flags, os.Stderr))
		case WithJSONHandler:
			l.handlers = append(l.handlers, newJSONHandler(l.level.Level(), flags, os.Stderr))
		default:
			l.handlers = append(l.handlers, newDefaultHandler(l.level.Level(), flags, os.Stderr))
		}
	}

	// set common record flags
	l.setFlags()

	return l, nil
}

// SystemDebug logs at logging.LevelSystemDebug.
func (l *Logger) SystemDebug(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelSystemDebug, msg, nil, args...)
}

// Debug logs at logging.LevelDebug.
func (l *Logger) Debug(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelDebug, msg, nil, args...)
}

// Info logs at logging.LevelInfo.
func (l *Logger) Info(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelInfo, msg, nil, args...)
}

// Task logs at logging.LevelTask and returns slog group with task info
// which can be used as logging attribute argument when logging task related events.
func (l *Logger) Task(name, description string) TaskInfo {
	t := TaskInfo{
		Name:        name,
		Description: description,
		startedAt:   time.Now(),
	}
	l.LogDepth(l.ctx, 1, LevelTask, t.Name, nil, slog.String("description", t.Description))
	return t
}

// Ok logs at logging.LevelOk.
func (l *Logger) Ok(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelOk, msg, nil, args...)
}

// Notice logs at logging.LevelNotice.
func (l *Logger) Notice(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelNotice, msg, nil, args...)
}

// Warn logs at logging.LevelWarn.
func (l *Logger) Warn(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelWarn, msg, nil, args...)
}

// NotImplemented logs at logging.LevelNotImplemented.
func (l *Logger) NotImplemented(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelNotImplemented, msg, nil, args...)
}

// Deprecated logs at logging.LevelDeprecated.
func (l *Logger) Deprecated(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelDeprecated, msg, nil, args...)
}

// Issue logs at logging.LevelIssue.
func (l *Logger) Issue(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelIssue, msg, nil, args...)
}

// Error logs at logging.LevelError.
func (l *Logger) Error(msg string, err error, args ...any) {
	l.LogDepth(l.ctx, 1, LevelError, msg, err, args...)
}

// BUG logs at logging.LevelBUG.
func (l *Logger) BUG(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelBUG, msg, nil, args...)
}

// Msg logs at logging.LevelAlways.
func (l *Logger) Msg(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelAlways, msg, nil, args...)
}

// Log logs at given log level.
func (l *Logger) Log(level Level, msg string, args ...any) {
	l.LogDepth(l.ctx, 1, level, msg, nil, args...)
}

func (l *Logger) HTTP(status int, path string, args ...any) {

}

// LogDepth is like Logger.Log, but accepts a call depth to adjust the file and line number
// in the log record. 1 refers to the caller of LogDepth; 2 refers to the caller's caller; and so on.
// Level argument ensures that logger ignores records whose level is lower however
// registered handlerer may ignore records if record level does not meet minimum level required
// by the handler.
func (l *Logger) LogDepth(ctx context.Context, calldepth int, level Level, msg string, err error, args ...any) {
	if l.level.Level() > level {
		return
	}

	rr := newRawRecord(time.Now().In(l.tloc), level, msg, err, args...)

	if l.flags.Has(WithRecordSource) {
		rr.setSource(calldepth)
	}

	r, err := rr.record(l.flags, l.tlay, l.replaceFuncs)
	if err != nil {
		l.error(err)
		return
	}
	for i := range l.handlers {
		if err := l.handlers[i].Handle(ctx, r); err != nil {
			l.error(err)
		}
	}
}

// Level returns current log level.
func (l *Logger) Level() Level {
	return l.level.Level()
}

// SetLevel sets level for logger.
func (l *Logger) SetLevel(level Level) {
	l.level.Set(level)
}

func (l *Logger) With(args ...any) *Logger {
	attrs := attrsFromArgs(args)
	l2 := l.clone()
	for _, h := range l.handlers {
		l2.handlers = append(l2.handlers, h.WithAttrs(attrs...))
	}
	return l2
}

func (l *Logger) WithGroup(group string) *Logger {
	l2 := l.clone()
	l2.handlers = nil
	for _, h := range l.handlers {
		l2.handlers = append(l2.handlers, h.WithGroup(group))
	}
	return l2
}

func (l *Logger) setFlags() {
	var (
		flags []RecordFlag = []RecordFlag{
			WithRecordTimestamp,
			WithRecordLevel,
			WithRecordMessage,
			WithRecordError,
			WithRecordData,
			WithRecordSource,
		}
		enabled RecordFlag
	)
	for _, h := range l.handlers {
		for _, flag := range flags {
			if h.Flags().Has(flag) && !enabled.Has(flag) {
				enabled |= flag
			}
		}
	}
	l.flags = enabled
}

func (l *Logger) clone() *Logger {
	l2 := &Logger{
		ctx:   context.Background(),
		level: &levelVar{},
	}
	l2.SetLevel(l.Level())

	tloc := *l.tloc
	l2.tloc = &tloc
	l2.tlay = l.tlay
	l2.replaceFuncs = slices.Clone(l.replaceFuncs)
	l2.flags = l.flags
	return l2
}

func (l *Logger) error(err error) {
	fmt.Fprintln(os.Stderr, err)
}
