// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"log/slog"
	"math"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected string
	}{
		{"LevelHappy", LevelHappy, levelHappyStr},
		{"LevelDebugPkg", LevelDebugPkg, levelDebugPkgStr},
		{"LevelDebugAddon", LevelDebugAddon, levelDebugAddonStr},
		{"LevelTrace", LevelTrace, levelTraceStr},
		{"LevelDebug", LevelDebug, levelDebugStr},
		{"LevelInfo", LevelInfo, levelInfoStr},
		{"LevelNotice", LevelNotice, levelNoticeStr},
		{"LevelSuccess", LevelSuccess, levelSuccessStr},
		{"LevelNotImpl", LevelNotImpl, levelNotImplStr},
		{"LevelWarn", LevelWarn, levelWarnStr},
		{"LevelDepr", LevelDepr, levelDeprStr},
		{"LevelError", LevelError, levelErrorStr},
		{"LevelBUG", LevelBUG, levelBugStr},
		{"LevelOut", LevelOut, levelOutStr},
		{"LevelQuiet", LevelQuiet, levelQuietStr},
		{"UnknownLevel", Level(999), slog.Level(999).String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.String()
			testutils.Equal(t, tt.expected, got, "Level.String() returned unexpected value")
		})
	}
}

func TestLevelFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Level
		err      bool
	}{
		{"Happy", levelHappyStr, LevelHappy, false},
		{"DebugPkg", levelDebugPkgStr, LevelDebugPkg, false},
		{"DebugAddon", levelDebugAddonStr, LevelDebugAddon, false},
		{"Trace", levelTraceStr, LevelTrace, false},
		{"Debug", levelDebugStr, LevelDebug, false},
		{"Info", levelInfoStr, LevelInfo, false},
		{"Notice", levelNoticeStr, LevelNotice, false},
		{"Success", levelSuccessStr, LevelSuccess, false},
		{"NotImpl", levelNotImplStr, LevelNotImpl, false},
		{"Warn", levelWarnStr, LevelWarn, false},
		{"Depr", levelDeprStr, LevelDepr, false},
		{"Error", levelErrorStr, LevelError, false},
		{"BUG", levelBugStr, LevelBUG, false},
		{"Out", levelOutStr, LevelOut, false},
		{"Quiet", levelQuietStr, LevelQuiet, false},
		{"Invalid", "invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LevelFromString(tt.input)
			if tt.err {
				testutils.Error(t, err, "LevelFromString(%q) expected error", tt.input)
				testutils.Equal(t, Level(0), got, "LevelFromString(%q) should return zero level on error", tt.input)
				return
			}
			testutils.NoError(t, err, "LevelFromString(%q) unexpected error", tt.input)
			testutils.Equal(t, tt.expected, got, "LevelFromString(%q) returned unexpected level", tt.input)
		})
	}
}

func TestLevelMarshalUnmarshalSetting(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		input    []byte
		expected Level
		err      bool
	}{
		{"Happy", LevelHappy, []byte(levelHappyStr), LevelHappy, false},
		{"Debug", LevelDebug, []byte(levelDebugStr), LevelDebug, false},
		{"Error", LevelError, []byte(levelErrorStr), LevelError, false},
		{"Invalid", Level(999), []byte("invalid"), Level(999), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test MarshalSetting
			gotBytes, err := tt.level.MarshalSetting()
			if tt.err {
				testutils.Error(t, err, "MarshalSetting() expected error, got %q", string(gotBytes))
			} else {
				testutils.NoError(t, err, "MarshalSetting() unexpected error")
				testutils.EqualAny(t, tt.input, gotBytes, "MarshalSetting() returned unexpected bytes")
			}

			// Test UnmarshalSetting
			var lvl Level
			err = lvl.UnmarshalSetting(tt.input)
			if tt.err {
				testutils.Error(t, err, "UnmarshalSetting(%q) expected error", tt.input)
			} else {
				testutils.NoError(t, err, "UnmarshalSetting(%q) unexpected error", tt.input)
				testutils.Equal(t, tt.expected, lvl, "UnmarshalSetting(%q) returned unexpected level", tt.input)
			}
		})
	}
}

func TestLevelSlogIntegration(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected slog.Level
	}{
		{"LevelHappy", LevelHappy, slog.Level(math.MinInt)},
		{"LevelDebug", LevelDebug, slog.LevelDebug},
		{"LevelInfo", LevelInfo, slog.LevelInfo},
		{"LevelWarn", LevelWarn, slog.LevelWarn},
		{"LevelError", LevelError, slog.LevelError},
		{"LevelOut", LevelOut, slog.Level(math.MaxInt - 2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.Level()
			testutils.Equal(t, tt.expected, got, "Level() returned unexpected slog.Level")
		})
	}
}

func TestLevelLogValue(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected slog.Value
	}{
		{"LevelHappy", LevelHappy, slog.StringValue(levelHappyStr)},
		{"LevelDebug", LevelDebug, slog.StringValue(levelDebugStr)},
		{"LevelError", LevelError, slog.StringValue(levelErrorStr)},
		{"UnknownLevel", Level(999), slog.StringValue(slog.Level(999).String())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.level.LogValue()
			testutils.EqualAny(t, tt.expected, got, "LogValue() returned unexpected slog.Value")
		})
	}
}
