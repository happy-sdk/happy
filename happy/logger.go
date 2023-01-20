// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package happy

import (
	"math"

	"golang.org/x/exp/slog"
)

type LogLevel int

type Level int8

const (
	// Happy Log Levels
	LogLevelSystemDebug    LogLevel = LogLevel(slog.LevelDebug - 1)
	LogLevelDebug          LogLevel = LogLevel(slog.LevelDebug)
	LogLevelInfo           LogLevel = LogLevel(slog.LevelInfo)
	LogLevelTask           LogLevel = LogLevel(slog.LevelInfo + 1)
	LogLevelOk             LogLevel = LogLevel(slog.LevelInfo + 2)
	LogLevelNotice         LogLevel = LogLevel(slog.LevelInfo + 3)
	LogLevelWarn           LogLevel = LogLevel(slog.LevelWarn)
	LogLevelNotImplemented LogLevel = LogLevel(slog.LevelWarn + 1)
	LogLevelDeprecated     LogLevel = LogLevel(slog.LevelWarn + 2)
	LogLevelIssue          LogLevel = LogLevel(slog.LevelWarn + 3)
	LogLevelError          LogLevel = LogLevel(slog.LevelError)
	LogLevelCritical       LogLevel = LogLevel(slog.LevelError + 1)
	LogLevelAlert          LogLevel = LogLevel(slog.LevelError + 2)
	LogLevelEmergency      LogLevel = LogLevel(slog.LevelError + 3)
	LogLevelBUG            LogLevel = 1000
	LogLevelAlways         LogLevel = math.MaxInt32
)
