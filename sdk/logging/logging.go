// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime"
	"time"
)

type Level int

const (
	levelSystemDebug    = slog.Level(math.MinInt)
	levelDebug          = slog.LevelDebug
	levelInfo           = slog.LevelInfo
	levelOk             = slog.Level(1)
	levelNotice         = slog.Level(2)
	levelNotImplemented = slog.Level(3)
	levelWarn           = slog.LevelWarn
	levelDeprecated     = slog.Level(5)
	levelError          = slog.LevelError
	levelBUG            = slog.Level(9)
	levelAlways         = slog.Level(math.MaxInt - 1)
	levelQuiet          = slog.Level(math.MaxInt)

	LevelSystemDebug    Level = Level(levelSystemDebug)
	LevelDebug          Level = Level(levelDebug)
	LevelInfo           Level = Level(levelInfo)
	LevelOk             Level = Level(levelOk)
	LevelNotice         Level = Level(levelNotice)
	LevelNotImplemented Level = Level(levelNotImplemented)
	LevelWarn           Level = Level(levelWarn)
	LevelDeprecated     Level = Level(levelDeprecated)
	LevelError          Level = Level(levelError)
	LevelBUG            Level = Level(levelBUG)
	LevelAlways         Level = Level(levelAlways)
	LevelQuiet          Level = Level(levelQuiet)

	levelSystemDebugStr    = "system"
	levelDebugStr          = "debug"
	levelInfoStr           = "info"
	levelOkStr             = "ok"
	levelNoticeStr         = "notice"
	levelNotImplementedStr = "notimpl"
	levelWarnStr           = "warn"
	levelDeprecatedStr     = "depr"
	levelErrorStr          = "error"
	levelBUGStr            = "bug"
	levelAlwaysStr         = ""
)

var lvlval = map[Level]string{
	LevelSystemDebug:    levelSystemDebugStr,
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

func (l Level) String() string {
	if str, ok := lvlval[l]; ok {
		return str
	}
	return slog.Level(l).String()
}

type Logger interface {
	SystemDebug(msg string, attrs ...slog.Attr)
	Debug(msg string, attrs ...slog.Attr)
	Info(msg string, attrs ...slog.Attr)
	Ok(msg string, attrs ...slog.Attr)
	Notice(msg string, attrs ...slog.Attr)
	NotImplemented(msg string, attrs ...slog.Attr)
	Warn(msg string, attrs ...slog.Attr)
	Deprecated(msg string, attrs ...slog.Attr)
	Error(msg string, attrs ...slog.Attr)
	BUG(msg string, attrs ...slog.Attr)
	Ln(msg string, attrs ...slog.Attr)
	HTTP(method, path string, status int, attrs ...slog.Attr)
	Handle(r slog.Record) error

	Enabled(lvl Level) bool
	Level() Level
	SetLevel(lvl Level)

	LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr)
}

type DefaultLogger struct {
	lvl     *slog.LevelVar
	log     *slog.Logger
	ctx     context.Context
	colored bool
}

func Default(lvl Level) *DefaultLogger {
	l := &DefaultLogger{
		lvl: new(slog.LevelVar),
		ctx: context.Background(),
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

func (l *DefaultLogger) SystemDebug(msg string, attrs ...slog.Attr) {
	l.logDepth(levelSystemDebug, msg, attrs...)
}

func (l *DefaultLogger) Debug(msg string, attrs ...slog.Attr) {
	l.logDepth(levelDebug, msg, attrs...)
}

func (l *DefaultLogger) Info(msg string, attrs ...slog.Attr) {
	l.logDepth(levelInfo, msg, attrs...)
}

func (l *DefaultLogger) Ok(msg string, attrs ...slog.Attr) {
	l.logDepth(levelOk, msg, attrs...)
}

func (l *DefaultLogger) Notice(msg string, attrs ...slog.Attr) {
	l.logDepth(levelNotice, msg, attrs...)
}

func (l *DefaultLogger) Warn(msg string, attrs ...slog.Attr) {
	l.logDepth(levelWarn, msg, attrs...)
}

func (l *DefaultLogger) NotImplemented(msg string, attrs ...slog.Attr) {
	l.logDepth(levelNotImplemented, msg, attrs...)
}

func (l *DefaultLogger) Deprecated(msg string, attrs ...slog.Attr) {
	l.logDepth(levelDeprecated, msg, attrs...)
}

func (l *DefaultLogger) Error(msg string, attrs ...slog.Attr) {
	l.logDepth(levelError, msg, attrs...)
}

func (l *DefaultLogger) BUG(msg string, attrs ...slog.Attr) {
	l.logDepth(levelBUG, msg, attrs...)
}

func (l *DefaultLogger) Ln(msg string, attrs ...slog.Attr) {
	l.logDepth(levelAlways, msg, attrs...)
}

func (l *DefaultLogger) HTTP(method, path string, status int, attrs ...slog.Attr) {
	switch status {
	case 100, 200:
		if l.log.Enabled(l.ctx, levelInfo) {
			l.http(method, path, status, attrs...)
		}
	case 300:
		if l.log.Enabled(l.ctx, levelWarn) {
			l.http(method, path, status, attrs...)
		}
	case 400:
		if l.log.Enabled(l.ctx, levelError) {
			l.http(method, path, status, attrs...)
		}
	case 500:
		if l.log.Enabled(l.ctx, levelError) {
			l.http(method, path, status, attrs...)
		}
	default:
		if l.log.Enabled(l.ctx, levelBUG) {
			attrs = append(attrs, slog.String("err", "invalid status code"))
			l.http(method, path, status, attrs...)
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
	runtime.Callers(depth, pcs[:])
	r := slog.NewRecord(time.Now(), slog.Level(lvl), msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = l.log.Handler().Handle(l.ctx, r)
}

func (l *DefaultLogger) Handle(r slog.Record) error {
	if !l.log.Enabled(l.ctx, r.Level) {
		return nil
	}
	return l.log.Handler().Handle(l.ctx, r)
}

func (l *DefaultLogger) http(method, path string, status int, attrs ...slog.Attr) {
	if ch, ok := l.log.Handler().(*ConsoleHandler); ok {
		ch.http(method, path, status, attrs...)
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), levelAlways, fmt.Sprintf("[%-8s %-3s] %s", method, fmt.Sprint(status), path), pcs[0])
	r.AddAttrs(attrs...)
	_ = l.log.Handler().Handle(l.ctx, r)

}

func (l *DefaultLogger) logDepth(lvl slog.Level, msg string, attrs ...slog.Attr) {
	if !l.log.Enabled(l.ctx, lvl) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), lvl, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = l.log.Handler().Handle(l.ctx, r)
}

type QueueRecord struct {
	pc    uintptr
	lvl   Level
	ts    time.Time
	msg   string
	attrs []slog.Attr
}

func NewQueueRecord(lvl Level, msg string, detph int, attrs ...slog.Attr) QueueRecord {
	var pcs [1]uintptr
	runtime.Callers(detph, pcs[:])

	return QueueRecord{
		lvl:   lvl,
		ts:    time.Now(),
		msg:   msg,
		attrs: attrs,
		pc:    pcs[0],
	}
}

func (qr QueueRecord) Record() slog.Record {
	r := slog.NewRecord(qr.ts, slog.Level(qr.lvl), qr.msg, qr.pc)
	r.AddAttrs(qr.attrs...)
	return r
}
