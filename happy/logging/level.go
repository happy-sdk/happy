// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/happy-sdk/vars"
	"golang.org/x/exp/slog"
)

type Level int32

const (
	LevelSystemDebug    Level = -10 // Level(slog.LevelDebug - 1)
	LevelDebug          Level = -4  // Level(slog.LevelDebug)
	LevelInfo           Level = 0   // Level(slog.LevelInfo)
	LevelTask           Level = 1   // Level(slog.LevelInfo + 1)
	LevelOk             Level = 2   // Level(slog.LevelInfo + 2)
	LevelNotice         Level = 3   // Level(slog.LevelInfo + 3)
	LevelWarn           Level = 4   // Level(slog.LevelWarn)
	LevelNotImplemented Level = 5   // Level(slog.LevelWarn + 1)
	LevelDeprecated     Level = 6   // Level(slog.LevelWarn + 2)
	LevelIssue          Level = 7   // Level(slog.LevelWarn + 3)
	LevelError          Level = 8   // Level(slog.LevelError)
	LevelBUG            Level = math.MaxInt32 - 2
	LevelAlways         Level = math.MaxInt32 - 1
	LevelQuiet          Level = math.MaxInt32
)

// String returns log level label string.
func (l Level) String() string {
	return l.value().String()
}

func (l Level) Int() int {
	return int(l)
}

// MarshalText implements encoding.TextMarshaler by calling Level.String.
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
// It accepts any string produced by [Level.MarshalText],
// ignoring case.
// It also accepts numeric offsets that would result in a different string on
// output. For example, "Error-8" would marshal as "INFO".
func (l *Level) UnmarshalText(data []byte) error {
	return l.parse(string(data))
}

// MarshalJSON implements [encoding/json.Marshaler]
// by quoting the output of [Level.String].
func (l Level) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, l.String()), nil
}

// UnmarshalJSON implements [encoding/json.Unmarshaler]
// It accepts any string produced by [Level.MarshalJSON],
// ignoring case.
// It also accepts numeric offsets that would result in a different string on
// output. For example, "Error-8" would marshal as "INFO".
func (l *Level) UnmarshalJSON(data []byte) error {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	return l.parse(s)
}

func (l *Level) parse(s string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("logging: level string %q: %w", s, err)
		}
	}()
	name := s
	offset := 0
	if i := strings.IndexAny(s, "+-"); i >= 0 {
		name = s[:i]
		offset, err = strconv.Atoi(s[i:])
		if err != nil {
			return err
		}
	}

	switch strings.ToLower(name) {
	case "system":
		*l = LevelSystemDebug
	case "debug":
		*l = LevelDebug
	case "info":
		*l = LevelInfo
	case "task":
		*l = LevelTask
	case "ok":
		*l = LevelOk
	case "notice":
		*l = LevelNotice
	case "warn":
		*l = LevelWarn
	case "notimpl":
		*l = LevelNotImplemented
	case "depr":
		*l = LevelDeprecated
	case "issue":
		*l = LevelIssue
	case "error":
		*l = LevelError
	case "bug":
		*l = LevelBUG
	case "msg":
		*l = LevelAlways
	case "quiet":
		*l = LevelQuiet
	default:
		return ErrUnknownLevelName
	}
	*l += Level(offset)
	return nil
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

var lvlval = map[Level]vars.Value{
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

var slogval = map[Level]slog.Value{
	LevelSystemDebug:    slog.StringValue(strSystemDebug),
	LevelDebug:          slog.StringValue(strDebug),
	LevelInfo:           slog.StringValue(strInfo),
	LevelTask:           slog.StringValue(strTask),
	LevelOk:             slog.StringValue(strOk),
	LevelNotice:         slog.StringValue(strNotice),
	LevelWarn:           slog.StringValue(strWarn),
	LevelNotImplemented: slog.StringValue(strNotImplemented),
	LevelDeprecated:     slog.StringValue(strDeprecated),
	LevelIssue:          slog.StringValue(strIssue),
	LevelError:          slog.StringValue(strError),
	LevelBUG:            slog.StringValue(strBUG),
	LevelAlways:         slog.StringValue(strAlways),
	LevelQuiet:          slog.StringValue(strQuiet),
}

func (l Level) slogValue() slog.Value {
	if v, ok := slogval[l]; ok {
		return v
	}
	if l > LevelError && l < LevelBUG {
		return slog.StringValue(fmt.Sprintf("error+%d", l-LevelError))
	}
	return slog.StringValue(fmt.Sprintf("system-%d", l+LevelSystemDebug))
}

func (l Level) value() vars.Value {
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

// A LevelVar is a Level variable, to allow a Handler level to change
// dynamically.
// It implements Leveler as well as a Set method,
// and it is safe for use by multiple goroutines.
// The zero LevelVar corresponds to LevelInfo.
type levelVar struct {
	val atomic.Int32
}

// Level returns v's level.
func (v *levelVar) Level() Level {
	return Level(int(v.val.Load()))
}

// Set sets log level.
func (v *levelVar) Set(l Level) {
	v.val.Store(int32(l))
}

// MarshalText implements [encoding.TextMarshaler]
// by calling [Level.MarshalText].
func (v *levelVar) MarshalText() ([]byte, error) {
	return v.Level().MarshalText()
}

// UnmarshalText implements [encoding.TextUnmarshaler]
// by calling [Level.UnmarshalText].
func (v *levelVar) UnmarshalText(data []byte) error {
	var l Level
	if err := l.UnmarshalText(data); err != nil {
		return err
	}
	v.Set(l)
	return nil
}
