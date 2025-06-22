// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/settings"
)

type Settings struct {
	Level           Level           `key:"level,config" default:"ok" mutation:"mutable" desc:"logging level"`
	WithSource      settings.Bool   `key:"with_source,config" default:"false" mutation:"once" desc:"Show source location in log messages"`
	TimestampFormat settings.String `key:"timeestamp_format,config" default:"15:04:05.000" mutation:"once" desc:"Timestamp format for log messages"`
	NoTimestamp     settings.Bool   `key:"no_timestamp,config" default:"false" mutation:"once" desc:"Do not show timestamps"`
	NoSlogDefault   settings.Bool   `key:"no_slog_default" default:"false" mutation:"once" desc:"Do not set the default slog logger"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Level int

const (
	lvlHappy          = slog.Level(math.MinInt)
	lvlHappyInit      = slog.Level(lvlHappy + 1)
	lvlDebug          = slog.LevelDebug
	lvlInfo           = slog.LevelInfo
	lvlNotice         = slog.Level(1)
	lvlOk             = slog.Level(2)
	lvlNotImplemented = slog.Level(3)
	lvlWarn           = slog.LevelWarn
	lvlDeprecated     = slog.Level(5)
	lvlError          = slog.LevelError
	lvlBUG            = slog.Level(9)
	lvlAlways         = slog.Level(math.MaxInt - 1)
	lvlQuiet          = slog.Level(math.MaxInt)

	levelHappy          Level = Level(lvlHappy)
	levelInit           Level = Level(lvlHappyInit)
	LevelDebug          Level = Level(lvlDebug)
	LevelInfo           Level = Level(lvlInfo)
	LevelNotice         Level = Level(lvlNotice)
	LevelOk             Level = Level(lvlOk)
	LevelNotImplemented Level = Level(lvlNotImplemented)
	LevelWarn           Level = Level(lvlWarn)
	LevelDeprecated     Level = Level(lvlDeprecated)
	LevelError          Level = Level(lvlError)
	LevelBUG            Level = Level(lvlBUG)
	LevelAlways         Level = Level(lvlAlways)
	LevelQuiet          Level = Level(lvlQuiet)

	levelHappyStr          = "happy"
	levelInitStr           = "happy:init"
	levelDebugStr          = "debug"
	levelInfoStr           = "info"
	levelNoticeStr         = "notice"
	levelOkStr             = "ok"
	levelNotImplementedStr = "notimpl"
	levelWarnStr           = "warn"
	levelDeprecatedStr     = "depr"
	levelErrorStr          = "error"
	levelBUGStr            = "bug"
	levelAlwaysStr         = "out"
)

var lvlval = map[Level]string{
	levelHappy:          levelHappyStr,
	levelInit:           levelInitStr,
	LevelDebug:          levelDebugStr,
	LevelInfo:           levelInfoStr,
	LevelOk:             levelOkStr,
	LevelNotice:         levelNoticeStr,
	LevelNotImplemented: levelNotImplementedStr,
	LevelWarn:           levelWarnStr,
	LevelDeprecated:     levelDeprecatedStr,
	LevelError:          levelErrorStr,
	LevelBUG:            levelBUGStr,
	LevelAlways:         levelAlwaysStr,
}

// String lookup table
var strToLvl = make(map[string]Level)

func init() {
	for level, str := range lvlval {
		strToLvl[str] = level
	}
}

func LevelFromString(levelStr string) (Level, error) {
	if level, ok := strToLvl[levelStr]; ok {
		return level, nil
	}
	return 0, fmt.Errorf("invalid level string %q", levelStr)
}

func (l Level) String() string {
	if str, ok := lvlval[l]; ok {
		return str
	}
	return slog.Level(l).String()
}

func (l Level) MarshalSetting() ([]byte, error) {
	// Simply cast the String to a byte slice.
	return []byte(l.String()), nil
}

func (l *Level) UnmarshalSetting(data []byte) error {
	// Directly convert the byte slice to String.
	lvl := string(data)
	for k, v := range lvlval {
		if v == lvl {
			*l = k
			return nil
		}
	}
	return nil
}

func (l Level) SettingKind() settings.Kind {
	return settings.KindString
}

type Logger interface {
	Debug(msg string, attrs ...slog.Attr)
	Info(msg string, attrs ...slog.Attr)
	Ok(msg string, attrs ...slog.Attr)
	Notice(msg string, attrs ...slog.Attr)
	NotImplemented(msg string, attrs ...slog.Attr)
	Warn(msg string, attrs ...slog.Attr)
	Deprecated(msg string, attrs ...slog.Attr)
	Error(msg string, attrs ...slog.Attr)
	Errors(err error, attrs ...slog.Attr)
	BUG(msg string, attrs ...slog.Attr)
	Println(msg string, attrs ...slog.Attr)
	Printf(format string, v ...any)
	HTTP(status int, method, path string, attrs ...slog.Attr)
	Handle(r slog.Record) error

	Enabled(lvl Level) bool
	Level() Level
	SetLevel(lvl Level)

	LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr)

	Logger() *slog.Logger

	ConsumeQueue(queue *QueueLogger) error
}

type DefaultLogger struct {
	tsloc *time.Location
	lvl   *slog.LevelVar
	log   *slog.Logger
	ctx   context.Context
}

func New(w io.Writer, lvl Level) *DefaultLogger {
	l := &DefaultLogger{
		lvl:   new(slog.LevelVar),
		ctx:   context.Background(),
		tsloc: time.Local,
	}

	l.lvl.Set(slog.Level(lvl))

	h := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: l.lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				a.Value = slog.StringValue(Level(level).String())
			}
			return a
		},
		AddSource: true,
	})
	l.log = slog.New(h)
	return l
}

func NewDefault(lvl Level) *DefaultLogger {
	l := &DefaultLogger{
		lvl:   new(slog.LevelVar),
		ctx:   context.Background(),
		tsloc: time.Local,
	}

	l.lvl.Set(slog.Level(lvl))

	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: l.lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				a.Value = slog.StringValue(Level(level).String())
			}
			return a
		},
		AddSource: true,
	})
	l.log = slog.New(h)
	return l
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

func (l *DefaultLogger) http(status int, method, path string, attrs ...slog.Attr) {
	if ch, ok := l.log.Handler().(*ConsoleHandler); ok {
		ch.http(status, method, path, attrs...)
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(l.ts(), lvlAlways, fmt.Sprintf("[%-8s %-3s] %s", method, fmt.Sprint(status), path), pcs[0])
	r.AddAttrs(attrs...)
	_ = l.log.Handler().Handle(l.ctx, r)

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

func (l *DefaultLogger) ts() time.Time {
	if l.tsloc == nil {
		panic("logging: time location is nil")
	}
	return time.Now().In(l.tsloc)
}
