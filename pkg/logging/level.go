// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"fmt"
	"log/slog"
	"math"
)

// Level represents a logging level, extending slog.Level with additional
// levels specific to the Happy SDK. It supports conversion to and from strings,
// marshaling for configuration settings, and integration with slog for structured
// logging. You can call Level.Level() to get the underlying slog.Level.
type Level slog.Level

// Predefined logging levels, ordered by severity from lowest to highest.
// Each level maps to a specific slog.Level value or a custom value for Happy SDK
// specific levels.
const (
	// LevelHappy is the lowest level, used for positive application events.
	LevelHappy Level = Level(math.MinInt)
	// LevelDebugPkg is for package-specific debug messages.
	LevelDebugPkg Level = LevelHappy + 1
	// LevelDebugAddon is for addon-specific debug messages.
	LevelDebugAddon Level = LevelDebugPkg + 1
	// LevelTrace is for detailed tracing, lower than standard debug.
	LevelTrace Level = Level(slog.LevelDebug - 4)
	// LevelDebug aligns with slog.LevelDebug for standard debugging.
	LevelDebug Level = Level(slog.LevelDebug)
	// LevelInfo aligns with slog.LevelInfo for informational messages.
	LevelInfo Level = Level(slog.LevelInfo)
	// LevelNotice is for notable events, slightly above Info.
	LevelNotice Level = Level(slog.LevelInfo + 1)
	// LevelSuccess indicates successful operations.
	LevelSuccess Level = Level(slog.LevelInfo + 2)
	// LevelNotImpl warns about unimplemented features.
	LevelNotImpl Level = Level(slog.LevelInfo + 3)
	// LevelWarn aligns with slog.LevelWarn for warnings.
	LevelWarn Level = Level(slog.LevelWarn)
	// LevelDepr indicates deprecated features or hot paths.
	LevelDepr Level = Level(slog.LevelWarn + 1)
	// LevelError aligns with slog.LevelError for error conditions.
	LevelError Level = Level(slog.LevelError)
	// LevelOut is for output that would typically go to stdout/stderr.
	// But application wants to log it e.g. having adapter which only listens to LevelOut.
	LevelOut Level = Level(math.MaxInt - 2)
	// LevelBUG indicates critical bugs in the application.
	LevelBUG Level = Level(math.MaxInt - 1)
	// LevelQuiet is the highest level, used to suppress all logging.
	// This level is not used by Adapters
	LevelQuiet Level = Level(math.MaxInt)

	levelHappyStr      = "happy"
	levelDebugPkgStr   = "pkg"
	levelDebugAddonStr = "addon"
	levelTraceStr      = "trace"
	levelDebugStr      = "debug"
	levelInfoStr       = "info"
	levelNoticeStr     = "notice"
	levelSuccessStr    = "ok"
	levelNotImplStr    = "notimpl"
	levelWarnStr       = "warn"
	levelDeprStr       = "deprecated"
	levelErrorStr      = "error"
	levelBugStr        = "bug"
	levelOutStr        = "out"
	levelQuietStr      = "quiet"
)

var lvlval = map[Level]string{
	LevelHappy:      levelHappyStr,
	LevelDebugPkg:   levelDebugPkgStr,
	LevelDebugAddon: levelDebugAddonStr,
	LevelTrace:      levelTraceStr,
	LevelDebug:      levelDebugStr,
	LevelInfo:       levelInfoStr,
	LevelNotice:     levelNoticeStr,
	LevelSuccess:    levelSuccessStr,
	LevelNotImpl:    levelNotImplStr,
	LevelWarn:       levelWarnStr,
	LevelDepr:       levelDeprStr,
	LevelError:      levelErrorStr,
	LevelBUG:        levelBugStr,
	LevelOut:        levelOutStr,
	LevelQuiet:      levelQuietStr,
}

var strToLvl = make(map[string]Level)

func init() {
	for level, str := range lvlval {
		strToLvl[str] = level
	}
}

// LevelFromString converts a string to a Level value. It returns an error if the
// string does not correspond to a defined level.
//
// Example:
//
//	lvl, err := LevelFromString("info")
//	if err != nil {
//	    // Handle error
//	}
//	// lvl is LevelInfo
func LevelFromString(levelStr string) (Level, error) {
	if level, ok := strToLvl[levelStr]; ok {
		return level, nil
	}
	return 0, fmt.Errorf("invalid level string %q", levelStr)
}

// String returns the string representation of the Level. For unrecognized levels,
// it falls back to the string representation of the underlying slog.Level.
func (l Level) String() string {
	if str, ok := lvlval[l]; ok {
		return str
	}
	return slog.Level(l).String()
}

// MarshalSetting serializes the Level to a byte slice for use in configuration
// settings. It returns an error if the level is not a recognized Happy SDK level.
// Used by settings package.
//
// Example:
//
//	b, err := LevelInfo.MarshalSetting()
//	if err != nil {
//	    // Handle error
//	}
//	// b is []byte("info")
func (l Level) MarshalSetting() ([]byte, error) {
	str := l.String()
	if _, ok := strToLvl[str]; ok {
		return []byte(str), nil
	}
	return []byte(str), fmt.Errorf("%w: invalid level %q", ErrLevel, l.String())
}

// UnmarshalSetting deserializes a byte slice into a Level value. It returns an
// error if the input does not correspond to a valid level.
// Used by settings package.
//
// Example:
//
//	var lvl Level
//	err := lvl.UnmarshalSetting([]byte("error"))
//	if err != nil {
//	    // Handle error
//	}
//	// lvl is LevelError
func (l *Level) UnmarshalSetting(data []byte) error {
	lvl, err := LevelFromString(string(data))
	if err != nil {
		return err
	}
	*l = lvl
	return nil
}

// Level converts the Level to an slog.Level for compatibility with the standard
// library's logging system.
func (l Level) Level() slog.Level { return slog.Level(l) }

// LogValue returns the Level as an slog.Value for use in structured logging.
// It uses the string representation of the level.
//
// Example:
//
//	logger.LogAttrs(context.Background(), LevelInfo, "message", slog.Any("level", LevelInfo.LogValue()))
func (l Level) LogValue() slog.Value {
	return slog.StringValue(l.String())
}
