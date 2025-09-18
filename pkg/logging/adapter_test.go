// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

type failingAdapter struct{}

func (fa *failingAdapter) Dispose() error {
	return errors.New("Dispose called")
}
func (fa *failingAdapter) Ready() {}
func (fa *failingAdapter) Flush() error {
	return errors.New("Flush called")
}

func (fa *failingAdapter) Enabled(context.Context, slog.Level) bool { return true }
func (fa *failingAdapter) Handle(context.Context, slog.Record) error {
	return errors.New("Handle called")
}
func (fa *failingAdapter) WithAttrs(attrs []slog.Attr) slog.Handler { return fa }
func (fa *failingAdapter) WithGroup(name string) slog.Handler       { return fa }

type failingHttpAdapter struct {
}

func (*failingHttpAdapter) Enabled(context.Context, slog.Level) bool { return true }
func (*failingHttpAdapter) Handle(context.Context, slog.Record) error {
	return errors.New("Handle called")
}
func (a *failingHttpAdapter) WithAttrs(attrs []slog.Attr) slog.Handler { return a }
func (a *failingHttpAdapter) WithGroup(name string) slog.Handler       { return a }

func (*failingHttpAdapter) HTTP(_ context.Context, method string, statusCode int, path string, record slog.Record) error {
	return errors.New("HTTP called")
}

type failingHttpBatchAdapter struct {
}

func (*failingHttpBatchAdapter) Enabled(context.Context, slog.Level) bool { return true }
func (*failingHttpBatchAdapter) Handle(context.Context, slog.Record) error {
	return errors.New("Handle called")
}
func (a *failingHttpBatchAdapter) WithAttrs(attrs []slog.Attr) slog.Handler { return a }
func (a *failingHttpBatchAdapter) WithGroup(name string) slog.Handler       { return a }
func (a *failingHttpBatchAdapter) HTTPBatchHandle(records []HttpRecord) error {
	return errors.New("HandleBatch called")
}

type httpBatchAdapter struct {
}

func (*httpBatchAdapter) Enabled(context.Context, slog.Level) bool     { return true }
func (*httpBatchAdapter) Handle(context.Context, slog.Record) error    { return nil }
func (a *httpBatchAdapter) WithAttrs(attrs []slog.Attr) slog.Handler   { return a }
func (a *httpBatchAdapter) WithGroup(name string) slog.Handler         { return a }
func (a *httpBatchAdapter) HTTPBatchHandle(records []HttpRecord) error { return nil }

func TestDefaultAdapterEnabled(t *testing.T) {
	logger, _, _ := newDefaultTestLogger()
	defer logger.Dispose()
	adapter := GetAdaptersFromHandler[*DefaultAdapter](logger.Handler())[0]

	tests := []struct {
		name     string
		level    slog.Level
		expected bool
	}{
		{"BelowThreshold", LevelDebug.Level(), false},
		{"AtThreshold", LevelInfo.Level(), true},
		{"AboveThreshold", LevelError.Level(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adapter.Enabled(context.Background(), tt.level)
			testutils.Equal(t, tt.expected, got, "Enabled() returned unexpected result")
		})
	}
}

func TestDefaultAdapterComposerBeforeCompose(t *testing.T) {
	adapterComposer := NewTextAdapter(NewBuffer())

	t.Run("Enabled", func(t *testing.T) {
		got := adapterComposer.Enabled(context.Background(), LevelInfo.Level())
		testutils.Equal(t, false, got, "Enabled() should return false before Compose")
	})

	t.Run("Handle", func(t *testing.T) {
		err := adapterComposer.Handle(context.Background(), slog.Record{Level: LevelInfo.Level()})
		testutils.Assert(t, errors.Is(err, ErrAdapterNotComposed), "Handle() should return ErrAdapterNotComposed")
	})

	t.Run("Err", func(t *testing.T) {
		err := adapterComposer.Err()
		testutils.Assert(t, errors.Is(err, ErrAdapterNotComposed), "Err() should return ErrAdapterNotComposed")
	})

	t.Run("WithAttrs", func(t *testing.T) {
		handler := adapterComposer.WithAttrs([]slog.Attr{slog.String("key", "value")})
		testutils.IsType(t, &AdapterComposer[*DefaultAdapter]{}, handler, "WithAttrs() should return same composer")
	})

	t.Run("WithGroup", func(t *testing.T) {
		handler := adapterComposer.WithGroup("group")
		testutils.IsType(t, &AdapterComposer[*DefaultAdapter]{}, handler, "WithGroup() should return same composer")
	})
}

func TestDefaultAdapterAllLogLevels(t *testing.T) {
	config := defaultTestConfig()
	config.Level = LevelHappy
	config.TimeFormat = "15:04:05"
	config.TimeLocation = time.UTC
	config.SetSlogOutput = false

	levels := []struct {
		level Level
		name  string
	}{
		{LevelHappy, "Happy"},
		{LevelTrace, "Trace"},
		{LevelDebug, "Debug"},
		{LevelInfo, "Info"},
		{LevelNotice, "Notice"},
		{LevelSuccess, "Success"},
		{LevelNotImpl, "NotImpl"},
		{LevelWarn, "Warn"},
		{LevelDepr, "Depr"},
		{LevelError, "Error"},
		{LevelOut, "Out"},
		{LevelBUG, "BUG"},
	}

	for _, lvl := range levels {
		t.Run(lvl.name, func(t *testing.T) {
			buf := NewBuffer()
			logger := New(config, NewTextAdapter(buf))
			adapter := GetAdaptersFromHandler[*DefaultAdapter](logger.Handler())[0]
			defer logger.Dispose()

			record := slog.Record{
				Level:   lvl.level.Level(),
				Message: "test message for " + lvl.name,
				Time:    time.Date(2025, 9, 6, 12, 0, 0, 0, time.UTC),
			}

			err := adapter.Handle(context.Background(), record)
			testutils.NoError(t, err, "Handle() unexpected error")
			logger.Flush()

			output := buf.String()
			testutils.Assert(t, strings.Contains(output, "test message for "+lvl.name), "Output should contain the message")
			testutils.Assert(t, strings.Contains(output, "12:00:00"), "Output should contain the timestamp")
		})
	}
}

func TestDefaultAdapterNoTimestamp(t *testing.T) {
	config := defaultTestConfig()
	config.Level = LevelInfo
	config.NoTimestamp = true

	logger, buf := newTestLogger(config)
	adapter := GetAdaptersFromHandler[*DefaultAdapter](logger.Handler())[0]
	defer logger.Dispose()

	record := slog.Record{
		Level:   LevelInfo.Level(),
		Message: "no timestamp test",
		Time:    time.Date(2025, 9, 6, 12, 0, 0, 0, time.UTC),
	}

	err := adapter.Handle(context.Background(), record)
	testutils.NoError(t, err, "Handle() unexpected error")
	adapter.Flush()

	output := buf.String()
	testutils.Assert(t, strings.Contains(output, "no timestamp test"), "Output should contain the message")
	testutils.Assert(t, !strings.Contains(output, "12:00:00"), "Output should not contain timestamp")
}

func TestDefaultAdapterConcurrentAccess(t *testing.T) {
	config := defaultTestConfig()
	config.Level = LevelInfo
	config.SetSlogOutput = false

	logger, buf := newTestLogger(config)

	adapter := GetAdaptersFromHandler[*DefaultAdapter](logger.Handler())[0]
	defer logger.Dispose()

	const numGoroutines = 10
	const recordsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	ctx := context.Background()

	// Launch multiple goroutines writing concurrently
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range recordsPerGoroutine {
				record := slog.Record{
					Level:   LevelInfo.Level(),
					Message: fmt.Sprintf("goroutine-%d-record-%d", id, j),
					Time:    time.Now(),
				}
				err := adapter.Handle(ctx, record)
				if err != nil {
					t.Errorf("Handle() unexpected error in goroutine %d: %v", id, err)
				}
			}
		}(i)
	}

	wg.Wait()
	adapter.Flush()

	output := buf.String()
	lineCount := strings.Count(output, "\n")
	testutils.Assert(t, lineCount == numGoroutines*recordsPerGoroutine,
		"Expected %d lines, got %d", numGoroutines*recordsPerGoroutine, lineCount)
}

func TestDefaultAdapterComplexAttributes(t *testing.T) {
	config := defaultTestConfig()
	config.Level = LevelInfo

	tests := []struct {
		name     string
		attrs    []slog.Attr
		expected string
	}{
		{
			name:     "MultipleStringAttrs",
			attrs:    []slog.Attr{slog.String("key1", "value1"), slog.String("key2", "value2")},
			expected: "key1=value1 key2=value2",
		},
		{
			name:     "MixedTypeAttrs",
			attrs:    []slog.Attr{slog.String("str", "value"), slog.Int("num", 42), slog.Bool("flag", true)},
			expected: "str=value num=42 flag=true",
		},
		{
			name:     "GroupAttribute",
			attrs:    []slog.Attr{slog.Group("group", slog.String("nested", "value"))},
			expected: "group.nested=value",
		},
		{
			name:     "NestedGroups",
			attrs:    []slog.Attr{slog.Group("outer", slog.Group("inner", slog.String("deep", "value")))},
			expected: "outer.inner.deep=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, buf := newTestLogger(config)
			adapter := GetAdaptersFromHandler[*DefaultAdapter](logger.Handler())[0]
			defer logger.Dispose()

			record := slog.Record{
				Level:   LevelInfo.Level(),
				Message: "test message",
				Time:    time.Date(2025, 9, 6, 12, 0, 0, 0, time.UTC),
			}
			record.AddAttrs(tt.attrs...)

			err := adapter.Handle(context.Background(), record)
			testutils.NoError(t, err, "Handle() unexpected error")
			adapter.Flush()
			output := buf.String()
			testutils.Assert(t, strings.Contains(output, tt.expected), "Output should contain expected ATTRS: got %s", output)
		})
	}
}

func TestNewAdapterWriterClose(t *testing.T) {
	a := NewAdapter(NewBuffer(), func(writer *Writer, config Config) slog.Handler {
		return slog.NewTextHandler(writer, &slog.HandlerOptions{})
	})
	logger := New(defaultTestConfig(), a)
	testutils.NoError(t, logger.Dispose())

	a2 := NewAdapter(NewWriter(NewBuffer()), func(writer *Writer, config Config) slog.Handler {
		return slog.NewTextHandler(writer, &slog.HandlerOptions{})
	})
	logger2 := New(defaultTestConfig(), a2)
	testutils.NoError(t, logger2.Dispose())
}

func TestFailingAdapter(t *testing.T) {
	a := NewAdapter(&closableTestWriter{}, func(writer *Writer, config Config) *failingAdapter {
		return &failingAdapter{}
	}).Compose(defaultTestConfig())
	logger := New(defaultTestConfig(), a)
	testutils.Error(t, logger.Dispose())
	testutils.Assert(t, !a.Enabled(context.TODO(), slog.LevelError))
}

func TestFailingHTTPAdapter(t *testing.T) {
	logger := New(defaultTestConfig(),
		&failingHttpAdapter{},
		&failingHttpBatchAdapter{},
		&httpBatchAdapter{},
	)

	defer logger.Dispose()

	rec := NewHttpRecord(
		time.Now(), http.MethodTrace, 201, "/home", "key", "value")
	rec2 := NewHttpRecord(
		time.Now(), http.MethodGet, 200, "/home", "key", "value")

	testutils.Error(t, logger.Handler().Handle(context.TODO(), rec))
	testutils.Error(t, logger.Handler().Handle(context.TODO(), rec2))
	logger.Flush()
}

func TestDerivedAdapter(t *testing.T) {
	logger, _, buf := newDefaultTestLogger()
	defer logger.Dispose()

	logger2 := logger.WithGroup("group1")
	logger2.Info("logger2 message", "key", "value")
	testutils.IsType(t, &derivedAdapter{}, logger2.Handler())
	logger3 := logger2.WithGroup("group2")
	testutils.IsType(t, &derivedAdapter{}, logger3.Handler())
	logger3.Info("logger3 message", "key", "value")

	logger4 := logger3.With("k2", "v2")
	testutils.IsType(t, &derivedAdapter{}, logger4.Handler())
	logger4.Info("logger4 message", "key", "value")
	logger5 := logger4.With("k3", "v3")
	testutils.IsType(t, &derivedAdapter{}, logger5.Handler())
	logger5.Info("logger5 message", "key", "value")

	line1 := buf.String()
	testutils.ContainsString(t, line1, "logger2 message")
	testutils.ContainsString(t, line1, "logger3 message")
	testutils.ContainsString(t, line1, "logger4 message")
	testutils.ContainsString(t, line1, "logger5 message")
	testutils.ContainsString(t, line1, "group1.key")
	testutils.ContainsString(t, line1, "group1.group2.key")
	testutils.ContainsString(t, line1, "value")
	testutils.ContainsString(t, line1, "group1.group2.k2")
	testutils.ContainsString(t, line1, "v2")
	testutils.ContainsString(t, line1, "group1.group2.k3")
	testutils.ContainsString(t, line1, "v3")

}

func TestDerivedDisposedAdapter(t *testing.T) {
	buf := NewBuffer()
	config := defaultTestConfig()
	config.NoTimestamp = true

	logger := New(config,
		NewTextAdapter(buf),
		&failingHttpAdapter{},
		&failingHttpBatchAdapter{},
		&failingAdapter{},
		&httpBatchAdapter{},
	)

	l2 := logger.
		WithGroup("g1").
		WithGroup("").
		WithGroup("g3").
		With().With("key", "val")

	l2.Handler().Handle(context.TODO(), NewHttpRecord(time.Now(), http.MethodGet, 200, "/"))
	l2.Info("logger message", "key", "value")

	testutils.IsType(t, &derivedAdapter{}, l2.Handler())

	h2 := l2.Handler().
		WithAttrs([]slog.Attr{}).WithAttrs([]slog.Attr{slog.String("key", "val")})

	testutils.EqualAny(t, h2, h2.WithGroup(""))
	logger.Dispose()

	testutils.ErrorIs(t, h2.Handle(context.TODO(), slog.Record{}), ErrAdapterDisposed)

	l3 := logger.
		WithGroup("g1").
		WithGroup("").
		WithGroup("g3")

	testutils.IsType(t, discardAdapter{}, l3.Handler())
	testutils.IsType(t, discardAdapter{}, h2.WithGroup("noop"))
	testutils.IsType(t, discardAdapter{}, h2.WithAttrs([]slog.Attr{slog.String("key", "val")}))
	line1 := buf.String()
	testutils.Equal(t, "level=info msg=\"logger message\" g1.g3.key=val g1.g3.key=value g1.g3.key=val\n", line1)
}

func TestReplaceAdaptersStdout(t *testing.T) {
	var (
		oldStdout = os.Stdout
	)
	defer func() {
		os.Stdout = oldStdout
	}()

	config := defaultTestConfig()
	config.NoTimestamp = true
	logger := New(config,
		NewTextAdapter(os.Stdout),
		NewTextAdapter(&testWriter{fail: true}),
		&failingAdapter{},
	)
	defer logger.Dispose()

	newStdout := NewBuffer()
	testutils.NoError(t, ReplaceAdaptersStdout(logger, newStdout))

	logger.Info("message")
	logger.Flush()

	testutils.Equal(t, "level=info msg=message\n", newStdout.String())
}

func TestReplaceAdaptersFailingStdout(t *testing.T) {
	var (
		oldStdout = os.Stdout
	)
	defer func() {
		os.Stdout = oldStdout
	}()

	tmpFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	os.Stdout = tmpFile

	config := defaultTestConfig()
	config.NoTimestamp = true
	logger := New(config,
		NewTextAdapter(os.Stdout),
		NewTextAdapter(&testWriter{fail: true}),
		&failingAdapter{},
	)
	defer logger.Dispose()

	h := logger.Handler().(*handler)
	testutils.Error(t, h.state.Load().adapters[1].replaceWriter(tmpFile))

	newStdout := NewBuffer()
	testutils.NoError(t, ReplaceAdaptersStdout(logger, newStdout))

	logger.Info("message")
	logger.Flush()

	testutils.Equal(t, "level=info msg=message\nlevel=info msg=message\n", newStdout.String())

	logger2 := New(config,
		NewTextAdapter(NewBuffer()),
	)
	testutils.ErrorIs(t, ReplaceAdaptersStdout(logger2, newStdout), ErrAdapterSwappingOutput)
}

func TestReplaceAdaptersStderr(t *testing.T) {
	var (
		oldStderr = os.Stderr
	)
	defer func() {
		os.Stderr = oldStderr
	}()

	config := defaultTestConfig()
	config.NoTimestamp = true
	logger := New(config,
		NewTextAdapter(os.Stderr),
		NewTextAdapter(&testWriter{fail: true}),
		&failingAdapter{},
	)
	defer logger.Dispose()

	newStderr := NewBuffer()
	testutils.NoError(t, ReplaceAdaptersStderr(logger, newStderr))

	logger.Info("message")
	logger.Flush()

	testutils.Equal(t, "level=info msg=message\n", newStderr.String())

	logger2 := New(config,
		NewTextAdapter(NewBuffer()),
	)
	testutils.ErrorIs(t, ReplaceAdaptersStderr(logger2, newStderr), ErrAdapterSwappingOutput)
}

func TestAdapterDispose(t *testing.T) {
	buf := NewBuffer()
	textad := NewTextAdapter(buf).Compose(defaultTestConfig()).(DisposableAdapter)
	jsonad := NewJSONAdapter(buf).Compose(defaultTestConfig()).(DisposableAdapter)
	testutils.NoError(t, textad.Dispose())
	testutils.NoError(t, textad.Dispose())
	testutils.NoError(t, jsonad.Dispose())
	testutils.NoError(t, jsonad.Dispose())

	testutils.IsType(t, DiscardAdapter, textad.WithGroup("group"))
	testutils.IsType(t, DiscardAdapter, textad.WithAttrs([]slog.Attr{slog.String("key", "value")}))
	testutils.IsType(t, DiscardAdapter, jsonad.WithGroup("group"))
	testutils.IsType(t, DiscardAdapter, jsonad.WithAttrs([]slog.Attr{slog.String("key", "value")}))

	textaf := textad.(FlushableAdapter)
	jsonaf := jsonad.(FlushableAdapter)
	testutils.ErrorIs(t, textaf.Flush(), ErrAdapterDisposed)
	testutils.ErrorIs(t, jsonaf.Flush(), ErrAdapterDisposed)

	logger := New(
		defaultTestConfig(),
		textad,
		jsonad,
	)
	testutils.NoError(t, logger.Dispose())
}

func TestAdapterComposerMisuse(t *testing.T) {
	a1 := &AdapterComposer[*slog.TextHandler]{}
	a2 := &AdapterComposer[*slog.TextHandler]{
		f: func(writer *Writer, config Config) *slog.TextHandler {
			return nil
		},
	}
	a3 := &AdapterComposer[*slog.TextHandler]{
		f: func(writer *Writer, config Config) *slog.TextHandler {
			return &slog.TextHandler{}
		},
	}

	config := defaultTestConfig()
	config.NoTimestamp = true
	logger := New(config, a1, a2)
	testutils.IsType(t, DiscardAdapter, logger.Handler())
	testutils.NoError(t, logger.Dispose())
	logger2 := New(config, a3)
	testutils.IsType(t, &handler{}, logger2.Handler())
	testutils.NoError(t, logger2.Dispose())
}

func TestAdapterFlush(t *testing.T) {
	config := defaultTestConfig()
	config.NoTimestamp = true
	logger := New(config,
		&failingAdapter{},
	)
	testutils.Error(t, logger.Dispose())
}
