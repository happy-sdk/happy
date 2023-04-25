// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"fmt"
	"math"
	"sync/atomic"

	"github.com/happy-sdk/vars"
)

type LogLevel int32

const (
	LevelSystemDebug    LogLevel = -10 // Level(slog.LevelDebug - 1)
	LevelDebug          LogLevel = -4  // Level(slog.LevelDebug)
	LevelInfo           LogLevel = 0   // Level(slog.LevelInfo)
	LevelTask           LogLevel = 1   // Level(slog.LevelInfo + 1)
	LevelOk             LogLevel = 2   // Level(slog.LevelInfo + 2)
	LevelNotice         LogLevel = 3   // Level(slog.LevelInfo + 3)
	LevelWarn           LogLevel = 4   // Level(slog.LevelWarn)
	LevelNotImplemented LogLevel = 5   // Level(slog.LevelWarn + 1)
	LevelDeprecated     LogLevel = 6   // Level(slog.LevelWarn + 2)
	LevelIssue          LogLevel = 7   // Level(slog.LevelWarn + 3)
	LevelError          LogLevel = 8   // Level(slog.LevelError)
	LevelBUG            LogLevel = math.MaxInt32 - 2
	LevelAlways         LogLevel = math.MaxInt32 - 1
	LevelQuiet          LogLevel = math.MaxInt32
)

func (l LogLevel) Int() int {
	return int(l)
}

// String returns log level label string.
func (l LogLevel) String() string {
	return l.value().String()
}

const (
	strSystemDebug    = "system"
	strDebug          = "debug"
	strInfo           = "info"
	strTask           = "task"
	strOk             = "ok"
	strNotice         = "notice"
	strWarn           = "warn"
	strNotImplemented = "notimpl"
	strDeprecated     = "depr"
	strIssue          = "issue"
	strError          = "error"
	strBUG            = "bug"
	strAlways         = "msg"
	strQuiet          = "quiet"
)

var lvlval = map[LogLevel]vars.Value{
	LevelSystemDebug:    vars.StringValue(strSystemDebug),
	LevelDebug:          vars.StringValue(strDebug),
	LevelInfo:           vars.StringValue(strInfo),
	LevelTask:           vars.StringValue(strTask),
	LevelOk:             vars.StringValue(strOk),
	LevelNotice:         vars.StringValue(strNotice),
	LevelWarn:           vars.StringValue(strWarn),
	LevelNotImplemented: vars.StringValue(strNotImplemented),
	LevelDeprecated:     vars.StringValue(strDeprecated),
	LevelIssue:          vars.StringValue(strIssue),
	LevelError:          vars.StringValue(strError),
	LevelBUG:            vars.StringValue(strBUG),
	LevelAlways:         vars.StringValue(strAlways),
	LevelQuiet:          vars.StringValue(strQuiet),
}

func (l LogLevel) value() vars.Value {
	if v, ok := lvlval[l]; ok {
		return v
	}

	if l > LevelError && l < LevelBUG {
		return vars.StringValue(fmt.Sprintf("error+%d", l-LevelError))
	}
	if l > LevelDebug && l < LevelInfo {
		return vars.StringValue(fmt.Sprintf("debug+%d", l-LevelDebug))
	}
	return vars.StringValue(fmt.Sprintf("system%d", l-LevelSystemDebug))
}

type LogLevelIface interface {
	String() string
	Int() int
}

type Logger[LVL LogLevelIface] interface {
	Level() LVL
	SetLevel(LVL)
}

type logger[LVL LogLevelIface] struct {
	level atomic.Pointer[LVL]
}

func (l *logger[LVL]) Level() LVL {
	lvl := l.level.Load()
	return *lvl
}

func (l *logger[LVL]) SetLevel(lvl LVL) {
	l.level.Store(&lvl)
}
