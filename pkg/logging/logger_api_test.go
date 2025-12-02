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

func TestLoggerConvenienceWithArgs(t *testing.T) {
	logger, ch := newCaptureLogger(t)

	// Test that args are properly converted and included
	logger.Happy("test message", "key1", "value1", "key2", 42, "key3", true)
	testutils.Equal(t, slog.Level(LevelHappy), ch.last.Level, "level mismatch")
	testutils.Equal(t, "test message", ch.last.Message, "message mismatch")

	// Verify attributes were added
	ch.last.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "key1":
			testutils.Equal(t, "value1", a.Value.String(), "key1 value mismatch")
		case "key2":
			testutils.Equal(t, int64(42), a.Value.Any().(int64), "key2 value mismatch")
		case "key3":
			testutils.Equal(t, true, a.Value.Any().(bool), "key3 value mismatch")
		}
		return true
	})
}

func TestLoggerConvenienceContextVariants(t *testing.T) {
	logger, ch := newCaptureLogger(t)

	ctx := context.WithValue(context.Background(), testContextKey, "value")

	type tc struct {
		name   string
		call   func()
		expect Level
	}

	tests := []tc{
		{"HappyContext", func() { logger.HappyContext(ctx, "x") }, LevelHappy},
		{"DebugPkgContext", func() { logger.DebugPkgContext(ctx, "x") }, LevelDebugPkg},
		{"DebugAddonContext", func() { logger.DebugAddonContext(ctx, "x") }, LevelDebugAddon},
		{"TraceContext", func() { logger.TraceContext(ctx, "x") }, LevelTrace},
		{"NoticeContext", func() { logger.NoticeContext(ctx, "x") }, LevelNotice},
		{"SuccessContext", func() { logger.SuccessContext(ctx, "x") }, LevelSuccess},
		{"NotImplContext", func() { logger.NotImplContext(ctx, "x") }, LevelNotImpl},
		{"NotImplementedContext", func() { logger.NotImplementedContext(ctx, "x") }, LevelNotImpl},
		{"DeprecatedContext", func() { logger.DeprecatedContext(ctx, "x") }, LevelDepr},
		{"OutContext", func() { logger.OutContext(ctx, "x") }, LevelOut},
		{"BUGContext", func() { logger.BUGContext(ctx, "x") }, LevelBUG},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.call()
			testutils.Equal(t, slog.Level(tt.expect), ch.last.Level, "level mismatch")
		})
	}
}

func TestLoggerConvenienceContextWithArgs(t *testing.T) {
	logger, ch := newCaptureLogger(t)
	ctx := context.WithValue(context.Background(), testContextKey, "value")

	// Test that Context variants work with args
	logger.HappyContext(ctx, "test message", "key1", "value1", "key2", 42)
	testutils.Equal(t, slog.Level(LevelHappy), ch.last.Level, "level mismatch")
	testutils.Equal(t, "test message", ch.last.Message, "message mismatch")

	// Verify attributes were added
	ch.last.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "key1":
			testutils.Equal(t, "value1", a.Value.String(), "key1 value mismatch")
		case "key2":
			testutils.Equal(t, int64(42), a.Value.Any().(int64), "key2 value mismatch")
		}
		return true
	})
}
