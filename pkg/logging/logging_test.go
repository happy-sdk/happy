// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging_test

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/logging/adapters"
)

type testWriter struct {
	fail bool
}

func (f *testWriter) Write(p []byte) (n int, err error) {
	if f.fail {
		return 0, errors.New("testWriter: write failed")
	}
	return len(p), nil
}

func TestAdapterError(t *testing.T) {
	err := logging.NewAdapterError(logging.ErrAdapter)
	err2 := logging.NewAdapterError(err)
	testutils.Equal(t, err, err2, "NewAdapterError should not wrap previous *AdapterError")
	err3 := logging.NewAdapterError(errors.New("some error"))
	testutils.ErrorIs(t, err3.Err(), logging.ErrAdapter, "non adapter errors should be wrapped with logging.ErrAdapter")
}

func TestLoggerWithSomeDisposedAdaptersSingleHandlererErr(t *testing.T) {
	errBuf := logging.NewBuffer()
	errorHandler := logging.NewAdapterWithHandler(errBuf, func(writer io.Writer, opts *slog.HandlerOptions) slog.Handler {
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: logging.LevelError.Level(),
		})
	})
	warnBuf := logging.NewBuffer()
	warnHandler := logging.NewAdapterWithHandler(warnBuf, func(writer io.Writer, opts *slog.HandlerOptions) slog.Handler {
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: logging.LevelWarn.Level(),
		})
	})
	infoBuf := logging.NewBuffer()

	config := logging.DefaultConfig()

	failingTextAdapter := logging.NewTextAdapter(&testWriter{fail: true}).Compose(config)
	failingConsoleAdapter := adapters.NewBufferedConsoleAdapter(&testWriter{fail: true}, adapters.ConsoleAdapterDefaultTheme())
	logger := logging.New(config,
		logging.DiscardAdapter,
		logging.NewTextAdapter(infoBuf),
		errorHandler,
		warnHandler,
		failingTextAdapter,
		failingConsoleAdapter,
	)
	defer logger.Dispose()
	testutils.NoError(t, logger.Handler().Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelDebug.Level(), "debug msg string", 0)))

	testutils.Equal(t,
		"[*slog.TextHandler] logging:writer:io: testWriter: write failed",
		logger.Handler().Handle(
			context.TODO(),
			slog.NewRecord(time.Now(), slog.LevelInfo, "info msg string", 0),
		).Error(),
	)
	testutils.Error(t, logger.Handler().Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelWarn.Level(), "warn msg string", 0)))
	testutils.Error(t, logger.Handler().Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelError.Level(), "error msg string", 0)))

	err := failingTextAdapter.Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelError.Level(), "error msg string", 0))

	if testutils.Error(t, err, "expected error") {
		testutils.Assert(t, errors.Is(failingTextAdapter.Handle(
			context.TODO(),
			slog.NewRecord(time.Now(), logging.LevelError.Level(), "error msg string", 0)),
			logging.ErrWriterIO,
		))
	}

	testutils.Error(t, logger.Dispose())

	testutils.Equal(t, 3, strings.Count(infoBuf.String(), "\n"), "info buffer")
	testutils.Equal(t, 2, strings.Count(warnBuf.String(), "\n"), "warn buffer")
	testutils.Equal(t, 1, strings.Count(errBuf.String(), "\n"), "error buffer")

	testutils.NoError(t, failingTextAdapter.Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelError.Level(),
			"error msg string", 0)), "logger is disposed handler should not error anymore")

}

func TestLoggerWithSomeDisposedAdaptersMultipleHandlererErr(t *testing.T) {
	errBuf := logging.NewBuffer()
	errorHandler := logging.NewAdapterWithHandler(errBuf, func(writer io.Writer, opts *slog.HandlerOptions) slog.Handler {
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: logging.LevelError.Level(),
		})
	})
	warnBuf := logging.NewBuffer()
	warnHandler := logging.NewAdapterWithHandler(warnBuf, func(writer io.Writer, opts *slog.HandlerOptions) slog.Handler {
		return slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: logging.LevelWarn.Level(),
		})
	})
	infoBuf := logging.NewBuffer()

	config := logging.DefaultConfig()

	failingTextAdapter := logging.NewTextAdapter(&testWriter{fail: true}).Compose(config)
	failingJSONAdapter := logging.NewJSONAdapter(&testWriter{fail: true}).Compose(config)
	failingConsoleAdapter := adapters.NewBufferedConsoleAdapter(&testWriter{fail: true}, adapters.ConsoleAdapterDefaultTheme())
	logger := logging.New(config,
		logging.DiscardAdapter,
		logging.NewTextAdapter(infoBuf),
		errorHandler,
		warnHandler,
		failingTextAdapter,
		failingJSONAdapter,
		failingConsoleAdapter,
	)
	testutils.NoError(t, logger.Handler().Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelDebug.Level(), "debug msg string", 0)))

	testutils.Equal(t,
		`logging:adapter: [*slog.TextHandler] logging:writer:io: testWriter: write failed
logging:adapter: [*slog.JSONHandler] logging:writer:io: testWriter: write failed`,
		logger.Handler().Handle(
			context.TODO(),
			slog.NewRecord(time.Now(), slog.LevelInfo, "info msg string", 0),
		).Error(),
	)
	testutils.Error(t, logger.Handler().Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelWarn.Level(), "warn msg string", 0)))
	testutils.Error(t, logger.Handler().Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelError.Level(), "error msg string", 0)))

	err := failingTextAdapter.Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelError.Level(), "error msg string", 0))
	if testutils.Error(t, err, "expected error") {
		testutils.Assert(t, errors.Is(failingTextAdapter.Handle(
			context.TODO(),
			slog.NewRecord(time.Now(), logging.LevelError.Level(), "error msg string", 0)),
			logging.ErrWriterIO,
		))
	}

	testutils.Error(t, logger.Dispose())

	testutils.Equal(t, 3, strings.Count(infoBuf.String(), "\n"), "info buffer")
	testutils.Equal(t, 2, strings.Count(warnBuf.String(), "\n"), "warn buffer")
	testutils.Equal(t, 1, strings.Count(errBuf.String(), "\n"), "error buffer")

	testutils.NoError(t, failingTextAdapter.Handle(
		context.TODO(),
		slog.NewRecord(time.Now(), logging.LevelError.Level(),
			"error msg string", 0)), "logger is disposed handler should not error anymore")
}

func TestRecordHTTPWithNonCompatibleAdapter(t *testing.T) {
	buf := logging.NewBuffer()
	text := logging.NewTextAdapter(buf)
	logger := logging.New(logging.DefaultConfig(), text)
	defer logger.Dispose()

	err := logger.Handler().Handle(context.TODO(), logging.NewHttpRecord(
		time.Now(), http.MethodTrace, 201, "/home", "key", "value"))
	testutils.NoError(t, err)
	testutils.Equal(t, "", buf.String())
}

func TestRecordHTTPWithCompatibleAdapter(t *testing.T) {
	buf := logging.NewBuffer()
	console := adapters.NewBufferedConsoleAdapter(buf, adapters.ConsoleAdapterDefaultTheme())
	logger := logging.New(logging.DefaultConfig(), console)

	defer logger.Dispose()

	rec := logging.NewHttpRecord(
		time.Now(), http.MethodTrace, 201, "/home", "key", "value")
	rec2 := logging.NewHttpRecord(
		time.Now(), http.MethodGet, 200, "/home", "key", "value")

	testutils.NoError(t, logger.Handler().Handle(context.TODO(), rec))
	testutils.NoError(t, logger.Handler().Handle(context.TODO(), rec2))
	logger.Flush()

	line := buf.String()
	testutils.Assert(t, line != "", "http log line must not be empty")

	testutils.ContainsString(t, line, http.MethodTrace)
	testutils.ContainsString(t, line, "201")
	testutils.ContainsString(t, line, "/home")
	testutils.ContainsString(t, line, "key")
	testutils.ContainsString(t, line, "value")

	testutils.ContainsString(t, line, http.MethodGet)
	testutils.ContainsString(t, line, "200")
}

func TestGetAdaptersFromHandler(t *testing.T) {
	testutils.Nil(t, logging.GetAdaptersFromHandler[logging.Adapter](nil))

	buf := logging.NewBuffer()
	buf2 := logging.NewBuffer()
	logger := logging.New(
		logging.DefaultConfig(),
		adapters.NewBufferedConsoleAdapter(buf, adapters.ConsoleAdapterDefaultTheme()),
		logging.NewAdapterWithHandler(buf2, func(writer io.Writer, opts *slog.HandlerOptions) *adapters.ConsoleAdapter {
			return &adapters.ConsoleAdapter{}
		}),
	)

	defer logger.Dispose()
	testutils.Nil(t, logging.GetAdaptersFromHandler[*adapters.ConsoleAdapter](nil))

	console := logging.GetAdaptersFromHandler[*adapters.ConsoleAdapter](logger.Handler())
	testutils.NotNil(t, console)
	testutils.Len(t, console, 1)
}

func TestQueueLogger(t *testing.T) {
	count := 100000
	queue := logging.NewQueueLogger(count)
	for range count {
		queue.Info("message")
	}

	config := logging.DefaultConfig()
	config.SetSlogOutput = false
	buf := logging.NewBuffer()
	text := logging.NewTextAdapter(buf)
	logger := logging.New(config, text)
	defer logger.Dispose()

	c, err := queue.Consume(logger)
	testutils.NoError(t, err)
	testutils.Equal(t, count, c, fmt.Sprintf("expected %d records", count))
	testutils.Equal(t, count, strings.Count(buf.String(), "\n"), fmt.Sprintf("expected %d records", count))
	testutils.NoError(t, queue.Dispose())
}

func TestReplaceSecrets(t *testing.T) {
	config := logging.DefaultConfig()
	config.SetSlogOutput = false
	config.Secrets = []string{"api_key"}

	buf1 := logging.NewBuffer()
	buf2 := logging.NewBuffer()
	logger := logging.New(config,
		logging.NewTextAdapter(buf1),
		logging.NewBufferedTextAdapter(buf2, nil),
	)
	logger2 := logger.WithGroup("group")
	logger3 := logger2.WithGroup("sub")
	defer logger.Dispose()

	logger.Info("message", "api_key", "secret_key", "key", "value")
	logger2.Info("message2", "api_key", "secret_key", "key", "value")
	logger3.Info("message3", "api_key", "secret_key", "key", "value")
	logger.Info("message4",
		slog.Group("our",
			slog.Group("domain",
				slog.String("api_key", "secret123"),
				slog.String("host", "localhost"),
			),
			slog.String("timeout", "30s"),
		),
	)
	testutils.NoError(t, logger.Flush())

	out1 := buf1.String()
	out2 := buf2.String()
	testutils.ContainsString(t, out1, "message")
	testutils.ContainsString(t, out2, "message")
	testutils.ContainsString(t, out1, "message2")
	testutils.ContainsString(t, out2, "message2")
	testutils.ContainsString(t, out1, "message3")
	testutils.ContainsString(t, out2, "message3")
	testutils.ContainsString(t, out1, "message4")
	testutils.ContainsString(t, out2, "message4")
	testutils.ContainsString(t, out1, "api_key")
	testutils.ContainsString(t, out2, "api_key")
	testutils.ContainsString(t, out1, "group.api_key")
	testutils.ContainsString(t, out2, "group.api_key")
	testutils.ContainsString(t, out1, "group.sub.api_key")
	testutils.ContainsString(t, out2, "group.sub.api_key")
	testutils.ContainsString(t, out1, "our.domain.host")
	testutils.ContainsString(t, out2, "our.domain.host")
	testutils.ContainsString(t, out1, "our.domain.api_key")
	testutils.ContainsString(t, out2, "our.domain.api_key")
	testutils.ContainsString(t, out1, "key")
	testutils.ContainsString(t, out2, "key")
	testutils.ContainsString(t, out1, "group.key")
	testutils.ContainsString(t, out2, "group.key")
	testutils.ContainsString(t, out1, "value")
	testutils.ContainsString(t, out2, "value")
	testutils.ContainsString(t, out1, "<redacted>")
	testutils.ContainsString(t, out2, "<redacted>")
	testutils.Assert(t, !strings.Contains(out1, "secret_key"))
	testutils.Assert(t, !strings.Contains(out2, "secret_key"))
}

func TestOmit(t *testing.T) {
	config := logging.DefaultConfig()
	config.SetSlogOutput = false
	config.Omit = []string{"api_key"}

	buf1 := logging.NewBuffer()
	buf2 := logging.NewBuffer()
	logger := logging.New(config,
		logging.NewTextAdapter(buf1),
		logging.NewBufferedTextAdapter(buf2, nil),
	)
	logger2 := logger.WithGroup("group")
	logger3 := logger2.WithGroup("sub")
	defer logger.Dispose()

	logger.Info("message", "api_key", "secret_key", "key", "value")
	logger2.Info("message2", "api_key", "secret_key", "key", "value")
	logger3.Info("message3", "api_key", "secret_key", "key", "value")
	logger.Info("message4",
		slog.Group("our",
			slog.Group("domain",
				slog.String("api_key", "secret123"),
				slog.String("host", "localhost"),
			),
			slog.String("timeout", "30s"),
		),
	)
	testutils.NoError(t, logger.Flush())

	out1 := buf1.String()
	out2 := buf2.String()
	testutils.ContainsString(t, out1, "message")
	testutils.ContainsString(t, out2, "message")
	testutils.ContainsString(t, out1, "message2")
	testutils.ContainsString(t, out2, "message2")
	testutils.ContainsString(t, out1, "message3")
	testutils.ContainsString(t, out2, "message3")
	testutils.ContainsString(t, out1, "message4")
	testutils.ContainsString(t, out2, "message4")
	testutils.ContainsString(t, out1, "key")
	testutils.ContainsString(t, out2, "key")
	testutils.ContainsString(t, out1, "group.key")
	testutils.ContainsString(t, out2, "group.key")
	testutils.ContainsString(t, out1, "group.sub.key")
	testutils.ContainsString(t, out2, "group.sub.key")
	testutils.ContainsString(t, out1, "our.domain.host")
	testutils.ContainsString(t, out2, "our.domain.host")
	testutils.ContainsString(t, out1, "value")
	testutils.ContainsString(t, out2, "value")

	testutils.Assert(t, !strings.Contains(out1, "api_key"))
	testutils.Assert(t, !strings.Contains(out2, "api_key"))
	testutils.Assert(t, !strings.Contains(out1, "group.api_key"))
	testutils.Assert(t, !strings.Contains(out2, "group.api_key"))
	testutils.Assert(t, !strings.Contains(out1, "group.sub.api_key"))
	testutils.Assert(t, !strings.Contains(out2, "group.sub.api_key"))
	testutils.Assert(t, !strings.Contains(out1, "our.domain.api_key"))
	testutils.Assert(t, !strings.Contains(out2, "our.domain.api_key"))
	testutils.Assert(t, !strings.Contains(out1, "secret_key"))
	testutils.Assert(t, !strings.Contains(out2, "secret_key"))
	testutils.Assert(t, !strings.Contains(out1, "<redacted>"))
	testutils.Assert(t, !strings.Contains(out2, "<redacted>"))
}

func TestProcessAttrsEmpty(t *testing.T) {
	config := logging.DefaultConfig()
	config.SetSlogOutput = false

	buf := logging.NewBuffer()
	logger := logging.New(config, logging.NewTextAdapter(buf))
	defer logger.Dispose()
	logger.With("key", "value").Info("message with no attributes")

	testutils.NoError(t, logger.Flush())

	out := buf.String()
	testutils.ContainsString(t, out, "message with no attributes")
}

func TestBufferedAdapterBatchWrites(t *testing.T) {
	t.Skip()
	config := logging.DefaultConfig()
	config.SetSlogOutput = false
	config.Adapter.BufferSize = 1024
	config.Adapter.BatchSize = 256

	dropped := &expvar.Int{}
	buf1 := logging.NewBuffer()
	// buf2 := logging.NewBuffer()
	// buf3 := logging.NewBuffer()
	logger := logging.New(config,
		logging.NewBufferedTextAdapter(buf1, dropped),
		// logging.NewBufferedJSONAdapter(buf2, nil),
		// adapters.NewBufferedConsoleAdapter(buf3, adapters.ConsoleAdapterDefaultTheme()),
	)
	defer logger.Dispose()

	count := 50000
	queue := logging.NewQueueLogger(count)
	queuel := queue.With("key", "value").WithGroup("group")
	for range count {
		queuel.Info("queue message")
		logger.Info("message")
	}

	consumed, err := queue.Consume(logger)
	logger.Flush()

	testutils.NoError(t, err)
	testutils.Equal(t, count, consumed, "expected %d", count)

	testutils.Assert(t, dropped.Value() == 0, "should not have any dropped records")
	testutils.Equal(t, count*2, strings.Count(buf1.String(), "\n"), "expected records")
	// testutils.Equal(t, count*2, strings.Count(buf2.String(), "\n"), "expected records")
	// testutils.Equal(t, count*2, strings.Count(buf3.String(), "\n"), "expected records")
}
