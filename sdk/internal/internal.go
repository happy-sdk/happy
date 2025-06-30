// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package internal

import (
	"fmt"
	"log/slog"
	"math"
	"runtime"

	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/sdk/events"
)

const (
	// these must be kept in sync with lvlHappy in logging/logging.go
	LogLevelHappy     logging.Level = logging.Level(math.MinInt)
	LogLevelHappyInit logging.Level = LogLevelHappy + 1
)

func Log(l logging.Logger, msg string, attrs ...slog.Attr) {
	LogDepth(l, 1, msg, attrs...)
}

func LogDepth(l logging.Logger, depth int, msg string, attrs ...slog.Attr) {
	l.LogDepth(depth+1, LogLevelHappy, msg, attrs...)
}

func LogInit(l logging.Logger, msg string, attrs ...slog.Attr) {
	LogInitDepth(l, 1, msg, attrs...)
}

func LogInitDepth(l logging.Logger, depth int, msg string, attrs ...slog.Attr) {
	l.LogDepth(depth+1, LogLevelHappyInit, msg, attrs...)
}

var TerminateSessionEvent = events.New("session", "terminate")

// RuntimeCallerStr is utility function to get the caller information.
// It returns the file and line number of the caller in form of string.
// e.g. /path/to/file.go:123
func RuntimeCallerStr(depth int) (string, bool) {
	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s:%d", file, line), true
}
