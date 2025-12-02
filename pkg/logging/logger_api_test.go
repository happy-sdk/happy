// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

// captureHandler records the last log record it sees.
type captureHandler struct {
	last slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.last = r.Clone()
	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(name string) slog.Handler {
	_ = name
	return h
}

// newCaptureLogger builds a Logger whose adapter uses captureHandler.
func newCaptureLogger(t *testing.T) (*Logger, *captureHandler) {
	t.Helper()

	ch := &captureHandler{}
	config := defaultTestConfig()
	// Allow all Happy levels for testing.
	config.Level = LevelHappy
	logger := New(
		config,
		NewAdapterWithHandler(&testWriter{}, func(_ io.Writer, _ *slog.HandlerOptions) slog.Handler {
			return ch
		}),
	)
	t.Cleanup(func() {
		_ = logger.Dispose()
	})
	return logger, ch
}

func TestLoggerConvenienceLevels(t *testing.T) {
	logger, ch := newCaptureLogger(t)

	type tc struct {
		name   string
		call   func()
		expect Level
	}

	tests := []tc{
		{"Happy", func() { logger.Happy("x") }, LevelHappy},
		{"DebugPkg", func() { logger.DebugPkg("x") }, LevelDebugPkg},
		{"DebugAddon", func() { logger.DebugAddon("x") }, LevelDebugAddon},
		{"Trace", func() { logger.Trace("x") }, LevelTrace},
		{"Notice", func() { logger.Notice("x") }, LevelNotice},
		{"Success", func() { logger.Success("x") }, LevelSuccess},
		{"NotImpl", func() { logger.NotImpl("x") }, LevelNotImpl},
		{"NotImplemented", func() { logger.NotImplemented("x") }, LevelNotImpl},
		{"Deprecated", func() { logger.Deprecated("x") }, LevelDepr},
		{"Out", func() { logger.Out("x") }, LevelOut},
		{"BUG", func() { logger.BUG("x") }, LevelBUG},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.call()
			testutils.Equal(t, slog.Level(tt.expect), ch.last.Level, "level mismatch")
		})
	}
}
