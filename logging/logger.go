// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"slices"
)

type DefaultLogger[LVL LevelIface] struct {
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
func New(config Config) (*DefaultLogger[Level], error) {
	l := &DefaultLogger[Level]{
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
func (l *DefaultLogger[LVL]) SystemDebug(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelSystemDebug, msg, nil, args...)
}

// Debug logs at logging.LevelDebug.
func (l *DefaultLogger[LVL]) Debug(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelDebug, msg, nil, args...)
}

// Info logs at logging.LevelInfo.
func (l *DefaultLogger[LVL]) Info(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelInfo, msg, nil, args...)
}

// Task logs at logging.LevelTask and returns slog group with task info
// which can be used as logging attribute argument when logging task related events.
func (l *DefaultLogger[LVL]) Task(name, description string) TaskInfo {
	l.Deprecated("Task is deprecated, use task pkg instead")
	t := TaskInfo{
		Name:        name,
		Description: description,
		startedAt:   time.Now(),
	}
	return t
}

// Ok logs at logging.LevelOk.
func (l *DefaultLogger[LVL]) Ok(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelOk, msg, nil, args...)
}

// Notice logs at logging.LevelNotice.
func (l *DefaultLogger[LVL]) Notice(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelNotice, msg, nil, args...)
}

// Warn logs at logging.LevelWarn.
func (l *DefaultLogger[LVL]) Warn(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelWarn, msg, nil, args...)
}

// NotImplemented logs at logging.LevelNotImplemented.
func (l *DefaultLogger[LVL]) NotImplemented(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelNotImplemented, msg, nil, args...)
}

// Deprecated logs at logging.LevelDeprecated.
func (l *DefaultLogger[LVL]) Deprecated(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelDeprecated, msg, nil, args...)
}

// Issue logs at logging.LevelIssue.
func (l *DefaultLogger[LVL]) Issue(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelIssue, msg, nil, args...)
}

// Error logs at logging.LevelError.
func (l *DefaultLogger[LVL]) Error(msg string, err error, args ...any) {
	l.LogDepth(l.ctx, 1, LevelError, msg, err, args...)
}

// BUG logs at logging.LevelBUG.
func (l *DefaultLogger[LVL]) BUG(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelBUG, msg, nil, args...)
}

// Msg logs at logging.LevelAlways.
func (l *DefaultLogger[LVL]) Msg(msg string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelAlways, msg, nil, args...)
}

// Msg logs at logging.LevelAlways.
func (l *DefaultLogger[LVL]) Msgf(format string, args ...any) {
	l.LogDepth(l.ctx, 1, LevelAlways, fmt.Sprintf(format, args...), nil, args...)
}

// Log logs at given log level.
func (l *DefaultLogger[LVL]) Log(level LevelIface, msg string, args ...any) {
	l.LogDepth(l.ctx, 1, level, msg, nil, args...)
}

func (l *DefaultLogger[LVL]) HTTP(status int, path string, args ...any) {

}

// LogDepth is like Logger.Log, but accepts a call depth to adjust the file and line number
// in the log record. 1 refers to the caller of LogDepth; 2 refers to the caller's caller; and so on.
// Level argument ensures that logger ignores records whose level is lower however
// registered handlerer may ignore records if record level does not meet minimum level required
// by the handler.
func (l *DefaultLogger[LVL]) LogDepth(ctx context.Context, calldepth int, lvl LevelIface, msg string, err error, args ...any) {
	level := Level(lvl.Int())
	if l.level.Level().Int() > level.Int() {
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
func (l *DefaultLogger[LVL]) Level() LevelIface {
	return l.level.Level()
}

// SetLevel sets level for logger.
func (l *DefaultLogger[LVL]) SetLevel(level LevelIface) {
	l.level.Set(level)
}

func (l *DefaultLogger[LVL]) With(args ...any) Logger {
	attrs := attrsFromArgs(args)
	l2 := l.clone()
	for _, h := range l.handlers {
		l2.handlers = append(l2.handlers, h.WithAttrs(attrs...))
	}
	return l2
}

func (l *DefaultLogger[LVL]) WithGroup(group string) Logger {
	l2 := l.clone()
	l2.handlers = nil
	for _, h := range l.handlers {
		l2.handlers = append(l2.handlers, h.WithGroup(group))
	}
	return l2
}

func (l *DefaultLogger[LVL]) setFlags() {
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

func (l *DefaultLogger[LVL]) clone() *DefaultLogger[LVL] {
	l2 := &DefaultLogger[LVL]{
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

func (l *DefaultLogger[LVL]) error(err error) {
	fmt.Fprintln(os.Stderr, err)
}

type Logger interface {
	Level() LevelIface
	SetLevel(LevelIface)
	SystemDebug(msg string, args ...any)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Ok(msg string, args ...any)
	Notice(msg string, args ...any)
	Warn(msg string, args ...any)
	NotImplemented(msg string, args ...any)
	Deprecated(msg string, args ...any)
	Issue(msg string, args ...any)
	Error(msg string, err error, args ...any)
	BUG(msg string, args ...any)
	Msg(msg string, args ...any)
	Msgf(format string, args ...any)
	Log(level LevelIface, msg string, args ...any)
	HTTP(status int, path string, args ...any)
	LogDepth(ctx context.Context, calldepth int, level LevelIface, msg string, err error, args ...any)
	With(args ...any) Logger
	WithGroup(group string) Logger
}
