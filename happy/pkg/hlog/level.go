// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package hlog

import (
	"fmt"
	"strconv"

	"golang.org/x/exp/slog"
)

const (
	// level mappings
	levelSystemDebug    = slog.LevelDebug - 1
	levelDebug          = slog.LevelDebug
	levelInfo           = slog.LevelInfo
	levelTask           = slog.LevelInfo + 1
	levelOk             = slog.LevelInfo + 2
	levelNotice         = slog.LevelInfo + 3
	levelWarn           = slog.LevelWarn
	levelNotImplemented = slog.LevelWarn + 1
	levelDeprecated     = slog.LevelWarn + 2
	levelIssue          = slog.LevelWarn + 3
	levelError          = slog.LevelError
	levelOut            = slog.LevelError + 1

	// Levels
	LevelSystemDebug    Level = Level(levelSystemDebug)
	LevelDebug          Level = Level(slog.LevelDebug)
	LevelInfo           Level = Level(levelInfo)
	LevelTask           Level = Level(levelTask)
	LevelOk             Level = Level(levelOk)
	LevelNotice         Level = Level(levelNotice)
	LevelWarn           Level = Level(levelWarn)
	LevelNotImplemented Level = Level(levelNotImplemented)
	LevelDeprecated     Level = Level(levelDeprecated)
	LevelIssue          Level = Level(levelIssue)
	LevelError          Level = Level(levelError)
	LevelOut            Level = Level(levelOut)
)

type Level slog.Level

// Level returns the receiver.
// It implements slog.Leveler.
func (l Level) Level() Level { return l }

func (l Level) String() string {
	switch l {
	case LevelSystemDebug:
		return "system"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelTask:
		return "task"
	case LevelOk:
		return "ok"
	case LevelNotice:
		return "notice"
	case LevelWarn:
		return "warn"
	case LevelNotImplemented:
		return "notimpl"
	case LevelDeprecated:
		return "depr"
	case LevelIssue:
		return "issue"
	case LevelError:
		return "error"
	case LevelOut:
		return "out"
	default:
		return slog.Level(l).String()
	}
}

func (l Level) Label() string {
	return fmt.Sprintf(" %-8s ", l.String())
}

func (l Level) ColorLabel() string {
	var (
		label  string
		fg, bg Color
	)
	switch l {
	case LevelSystemDebug:
		label = "system"
		fg, bg = FgBlue, 0
	case LevelDebug:
		label = "debug"
		fg, bg = FgWhite, 0
	case LevelInfo:
		label = "info"
		fg, bg = FgCyan, 0
	case LevelTask:
		label = "task"
		fg, bg = FgBlue, BgBlack
	case LevelOk:
		label = "ok"
		fg, bg = FgGreen, 0

	case LevelNotice:
		label = "notice"
		fg, bg = FgBlack, BgCyan
	case LevelWarn:
		label = "warn"
		fg, bg = FgYellow, 0
	case LevelNotImplemented:
		label = "notimpl"
		fg, bg = FgYellow, 0
	case LevelDeprecated:
		label = "depr"
		fg, bg = FgYellow, 0
	case LevelIssue:
		label = "issue"
		fg, bg = FgBlack, BgYellow
	case LevelError:
		label = "error"
		fg, bg = FgRed, BgBlack
	case LevelOut:
		label = "out"
		fg, bg = FgWhite, 0
	default:
		label = slog.Level(l).String()
		fg, bg = FgRed, BgBlack
	}
	return Colorize(fmt.Sprintf("%-8s", label), fg, bg, 1)
}

func (l Level) MarshalJSON() ([]byte, error) {
	// AppendQuote is sufficient for JSON-encoding all Level strings.
	// They don't contain any runes that would produce invalid JSON
	// when escaped.
	return strconv.AppendQuote(nil, l.String()), nil
}

func (l Level) color() (start []byte) {
	start = []byte{'\033', '['}
	var fg Color
	switch l {
	case LevelSystemDebug:
		fg = FgBlue
	case LevelDebug:
		fg = FgWhite
	case LevelInfo:
		fg = FgCyan
	case LevelTask:
		fg = FgBlue
	case LevelOk:
		fg = FgGreen

	case LevelNotice:
		fg = FgBlack
	case LevelWarn:
		fg = FgYellow
	case LevelNotImplemented:
		fg = FgYellow
	case LevelDeprecated:
		fg = FgYellow
	case LevelIssue:
		fg = FgBlack
	case LevelError:
		fg = FgRed
	case LevelOut:
		fg = FgWhite
	default:
		fg = FgRed
	}
	switch fgc := (fg & fgMask) >> fgShift; {
	case fgc <= 7:
		// '3' and the value itself
		start = append(start, '3', '0'+byte(fgc))
	case fg <= 15:
		// '9' and the value itself
		start = append(start, '9', '0'+byte(fgc&^0x08)) // clear bright flag
	default:
		start = append(start, '3', '8', ';', '5', ';')
		start = append(start, coloritoa(byte(fgc))...)
	}
	start = append(start, 'm')
	return
}
