// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package logging

import (
	"fmt"
	"log/slog"
	"math"
)

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
