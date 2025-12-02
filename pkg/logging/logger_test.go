// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

type testContextKeyType string

var testContextKey = testContextKeyType("test")

var errLoggingTest = errors.New("logging:test")

func defaultTestConfig() Config {
	config := DefaultConfig()
	config.SetSlogOutput = false
	return config
}

func newTestLogger(config Config) (*Logger, *Buffer) {
	buf := NewBuffer()
	logger := New(config, NewTextAdapter(buf))
	return logger, buf
}

func newDefaultTestLogger() (*Logger, Config, *Buffer) {
	config := defaultTestConfig()
	logger, buf := newTestLogger(config)
	return logger, config, buf
}

type closableTestWriter struct {
	testWriter
}

func (w *closableTestWriter) Close() error {
	return errors.New("closableTestWriter.Close")
}

func (w *closableTestWriter) Sync() error {
	return errors.New("closableTestWriter.Sync")
}

func TestLogger(t *testing.T) {
	args := []any{
		"key1", "value1",
		"key2", 42,
		"key3", true,
		"key4", "value4",
	}

	config := defaultTestConfig()
	config.Level = LevelHappy

	slogout := NewBuffer()
	slogLogger := slog.New(slog.NewTextHandler(slogout, config.HandlerOptions()))
	for range 10000 {

		slogLogger.Log(t.Context(), LevelHappy.Level(), "log message LevelHappy", args...)
		slogLogger.Log(t.Context(), LevelDebugPkg.Level(), "log message LevelDebugPkg", args...)
		slogLogger.Log(t.Context(), LevelDebugAddon.Level(), "log message LevelDebugAddon", args...)
		slogLogger.Log(t.Context(), LevelDebugAddon.Level(), "log message LevelDebugAddon", args...)
		slogLogger.Log(t.Context(), LevelTrace.Level(), "log message LevelTrace", args...)
		slogLogger.Debug("log message LevelDebug", args...)
		slogLogger.Info("log message LevelInfo", args...)
		slogLogger.Log(t.Context(), LevelNotice.Level(), "log message LevelNotice", args...)
		slogLogger.Log(t.Context(), LevelSuccess.Level(), "log message LevelSuccess", args...)
		slogLogger.Log(t.Context(), LevelNotImpl.Level(), "log message LevelNotImpl", args...)
		slogLogger.Warn("log message LevelWarn", args...)
		slogLogger.Log(t.Context(), LevelDepr.Level(), "log message LevelDepr", args...)
		slogLogger.Error("log message LevelError", args...)
		slogLogger.Log(t.Context(), LevelOut.Level(), "log message LevelOut", args...)
		slogLogger.Log(t.Context(), LevelBUG.Level(), "log message LevelBUG", args...)
	}
	logout := NewBuffer()

	logger := New(
		config,
		NewAdapterWithHandler(logout, slog.NewTextHandler),
	)
	defer func() {
		testutils.NoError(t, logger.Dispose())
	}()

	for range 10000 {
		logger.Log(t.Context(), LevelHappy.Level(), "log message LevelHappy", args...)
		logger.Log(t.Context(), LevelDebugPkg.Level(), "log message LevelDebugPkg", args...)
		logger.Log(t.Context(), LevelDebugAddon.Level(), "log message LevelDebugAddon", args...)
		logger.Log(t.Context(), LevelDebugAddon.Level(), "log message LevelDebugAddon", args...)
		logger.Log(t.Context(), LevelTrace.Level(), "log message LevelTrace", args...)
		logger.Debug("log message LevelDebug", args...)
		logger.Info("log message LevelInfo", args...)
		logger.Log(t.Context(), LevelNotice.Level(), "log message LevelNotice", args...)
		logger.Log(t.Context(), LevelSuccess.Level(), "log message LevelSuccess", args...)
		logger.Log(t.Context(), LevelNotImpl.Level(), "log message LevelNotImpl", args...)
		logger.Warn("log message LevelWarn", args...)
		logger.Log(t.Context(), LevelDepr.Level(), "log message LevelDepr", args...)
		logger.Error("log message LevelError", args...)
		logger.Log(t.Context(), LevelOut.Level(), "log message LevelOut", args...)
		logger.Log(t.Context(), LevelBUG.Level(), "log message LevelBUG", args...)
	}
	testutils.NoError(t, logger.Flush())

	testutils.Equal(t, 15230000, slogout.buf.Len(), "slog buffer should be creater than 0")
	testutils.Equal(t, 15230000, logout.buf.Len(), "Logger buffer should be creater than 0")
	testutils.Equal(t, slogout.buf.Len(), logout.buf.Len(), "both log buffer should be same length")
}

type testWriter struct {
	fail bool
}

func (f *testWriter) Write(p []byte) (n int, err error) {
	if f.fail {
		return 0, errors.New("testWriter: write failed")
	}
	return len(p), nil
}

func (f *testWriter) Close() error {
	if f.fail {
		return fmt.Errorf("%w: testWriter: close failed", errLoggingTest)
	}
	return nil
}

func TestLoggerDefaultAdapter(t *testing.T) {
	config := defaultTestConfig()
	config.SetSlogOutput = true
	logger := New(config)
	testutils.NotNil(t, logger, "logger must be created wth discard adapter")
	logger.SetLevel(LevelInfo)
	testutils.NoError(t, logger.Flush(), "Default Logger Flush should not error")

	if testutils.IsType(t, &handler{}, logger.Handler(), "expected *handler") {
		handler := logger.Handler().(*handler)
		adapters := handler.state.Load().adapters
		testutils.NotNil(t, adapters, "adapters should not be nil")
		if testutils.Len(t, adapters, 1, "expected default logger to have 1 adapter") {
			testutils.IsType(t, adapters[0].handler, &DefaultAdapter{}, "expected *DefaultAdapter")
		}
	}

	workingHandler := logger.Handler().(*handler)
	testutils.NoError(t, workingHandler.Flush())

	testutils.NoError(t, logger.Dispose(), "Default Logger Dispose should not error")

	testutils.IsType(t, discardAdapter{}, logger.Handler(), "expected discardAdapter")
	testutils.IsType(t, discardAdapter{}, logger.Handler().WithGroup("group"), "expected discardAdapter")
	testutils.IsType(t, discardAdapter{}, logger.Handler().WithAttrs([]slog.Attr{}), "expected discardAdapter")
	disposedHandler := logger.Handler().(discardAdapter)

	testutils.Assert(t, !workingHandler.Enabled(context.TODO(), LevelInfo.Level()), "Logger should report false for any level after disposed")
	testutils.ErrorIs(t, workingHandler.Handle(context.TODO(), slog.Record{}), ErrLoggerDisposed)
	testutils.ErrorIs(t, workingHandler.Flush(), ErrLoggerDisposed)
	testutils.IsType(t, discardAdapter{}, workingHandler.WithGroup("group"), "expected discardAdapter")
	testutils.IsType(t, discardAdapter{}, workingHandler.WithAttrs([]slog.Attr{}), "expected discardAdapter")

	testutils.Assert(t, !disposedHandler.Enabled(context.TODO(), LevelInfo.Level()), "Logger should report false for any level after disposed")
}

func TestLoggerDefaultAdapterNil(t *testing.T) {
	config := defaultTestConfig()
	config.SetSlogOutput = true
	logger := New(config, nil)

	testutils.NotNil(t, logger, "logger must be created wth discard adapter")
	logger.SetLevel(LevelInfo)
	testutils.ErrorIs(t, logger.Flush(), ErrLoggerDisposed, "Default Logger Flush should error when provided nil adapter")

	testutils.IsType(t, DiscardAdapter, logger.Handler(), "expected *handler")

	testutils.NoError(t, logger.Dispose(), "Default Logger Dispose should not error")

	testutils.IsType(t, discardAdapter{}, logger.Handler(), "expected discardAdapter")
	testutils.IsType(t, discardAdapter{}, logger.Handler().WithGroup("group"), "expected discardAdapter")
	testutils.IsType(t, discardAdapter{}, logger.Handler().WithAttrs([]slog.Attr{}), "expected discardAdapter")
}
