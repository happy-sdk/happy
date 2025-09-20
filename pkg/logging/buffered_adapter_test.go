// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func newBufferedLogger(config Config, droppedRecords *expvar.Int) (*Logger, *Buffer) {
	buf := NewBuffer()
	logger := New(config,
		NewBufferedTextAdapter(buf, droppedRecords),
		NewBufferedJSONAdapter(buf, droppedRecords),
		&httpBatchAdapter{},
	)
	return logger, buf
}

func TestBufferedAdapterHandle(t *testing.T) {
	tests := []struct {
		name             string
		config           Config
		record           slog.Record
		ctx              context.Context
		expectError      bool
		shouldContain    []string // Things that should be present
		shouldNotContain []string // Things that should not be present
	}{
		{
			name: "BasicInfoLog",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    1024,
					Policy:        AdapterPolicyBlock,
					BatchSize:     DefaultAdapterBatchSize,
					FlushInterval: DefaultAdapterFlushInterval,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			record: slog.Record{
				Level:   LevelInfo.Level(),
				Message: "test message",
				Time:    time.Date(2025, 9, 6, 12, 0, 0, 0, time.UTC),
			},
			ctx:           context.Background(),
			expectError:   false,
			shouldContain: []string{"info", "12:00:00.000", "test message"},
		},
		{
			name: "WithAttributes",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    1024,
					Policy:        AdapterPolicyBlock,
					BatchSize:     DefaultAdapterBatchSize,
					FlushInterval: DefaultAdapterFlushInterval,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			record: func() slog.Record {
				r := slog.Record{
					Level:   LevelInfo.Level(),
					Message: "test message",
					Time:    time.Date(2025, 9, 6, 12, 0, 0, 0, time.UTC),
				}
				r.AddAttrs(slog.String("key", "value"))
				return r
			}(),
			ctx:           context.Background(),
			expectError:   false,
			shouldContain: []string{"info", "12:00:00.000", "test message", "key=value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, buf := newBufferedLogger(tt.config, nil)
			defer func() {
				testutils.NoError(t, logger.Dispose())
			}()
			err := logger.Handler().Handle(tt.ctx, tt.record)
			if tt.expectError {
				testutils.Error(t, err, "Handle() expected error")
				return
			}
			testutils.NoError(t, err, "Handle() unexpected error")
			testutils.NoError(t, logger.Flush())

			got := buf.String()

			// Check that expected substrings are present
			for _, substr := range tt.shouldContain {
				testutils.Assert(t, strings.Contains(got, substr),
					"Output should contain '%s', got: %q", substr, got)
			}

			// Check that unwanted substrings are not present
			for _, substr := range tt.shouldNotContain {
				testutils.Assert(t, !strings.Contains(got, substr),
					"Output should not contain '%s', got: %q", substr, got)
			}

			// Verify output ends with newline
			testutils.Assert(t, strings.HasSuffix(got, "\n"), "Output should end with newline")
		})
	}
}

func TestBufferedAdapterMultipleDispose(t *testing.T) {
	logger, _ := newBufferedLogger(defaultTestConfig(), nil)
	defer func() {
		testutils.NoError(t, logger.Dispose())
	}()

	// First dispose
	err1 := logger.Dispose()
	testutils.NoError(t, err1, "First Dispose() should not error")

	// Second dispose
	err2 := logger.Dispose()
	// Should return the last error or nil, but not panic
	_ = err2 // Accept any result from second dispose

	// Verify we can still call methods without panic
	enabled := logger.Handler().Enabled(context.Background(), LevelInfo.Level())
	testutils.Equal(t, false, enabled, "Enabled() should return false after dispose")
}

func TestBufferedAdapterWithAttrsCloning(t *testing.T) {
	logger, _ := newBufferedLogger(defaultTestConfig(), nil)
	defer func() {
		testutils.NoError(t, logger.Dispose())
	}()

	// Create new handler with attributes
	attrs := []slog.Attr{slog.String("component", "test")}
	newHandler := logger.Handler().WithAttrs(attrs)

	// Both should be functional
	record := slog.Record{
		Level:   LevelInfo.Level(),
		Message: "test message",
		Time:    time.Now(),
	}

	err1 := logger.Handler().Handle(context.Background(), record)
	testutils.NoError(t, err1, "Original adapter should work")

	err2 := newHandler.Handle(context.Background(), record)
	testutils.NoError(t, err2, "New adapter should work")
}

func TestBufferedAdapterContextCancellation(t *testing.T) {
	t.Run("PolicyBlock", func(t *testing.T) {
		config := defaultTestConfig()
		config.Level = LevelInfo
		config.Adapter.Policy = AdapterPolicyBlock
		config.SetSlogOutput = false

		adapterI := NewBufferedTextAdapter(NewBuffer(), nil).Compose(config)
		logger := New(config, adapterI)
		adapter := adapterI.(*adapter)
		bufadapter := adapter.handler.(*BufferedAdapter[Adapter])

		defer func() {
			testutils.NoError(t, logger.Dispose())
		}()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		time.Sleep(time.Microsecond * 10)

		record := slog.Record{
			Level:   LevelInfo.Level(),
			Message: "cancelled context test",
			Time:    time.Now(),
		}

		testutils.NoError(t, bufadapter.Handle(ctx, record), "Handle() should not return error for cancelled context")
	})
	t.Run("PolicyDrop", func(t *testing.T) {
		config := defaultTestConfig()
		config.Level = LevelInfo
		config.Adapter.Policy = AdapterPolicyDrop
		config.SetSlogOutput = false

		adapterI := NewBufferedTextAdapter(NewBuffer(), nil).Compose(config)
		logger := New(config, adapterI)
		adapter := adapterI.(*adapter)
		bufadapter := adapter.handler.(*BufferedAdapter[Adapter])

		defer func() {
			testutils.NoError(t, logger.Dispose())
		}()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		time.Sleep(time.Microsecond * 10)

		record := slog.Record{
			Level:   LevelInfo.Level(),
			Message: "cancelled context test",
			Time:    time.Now(),
		}

		testutils.NoError(t, bufadapter.Handle(ctx, record), "Handle() should not return error for cancelled context")
	})
}

func TestBufferedAdapterErrorInHandle(t *testing.T) {
	config := defaultTestConfig()

	// Use a writer that will fail
	failingWriter := &testWriter{fail: true}
	adapterI := NewBufferedTextAdapter(failingWriter, nil).Compose(config)
	logger := New(config, adapterI)
	adapter := adapterI.(*adapter)
	bufadapter := adapter.handler.(*BufferedAdapter[Adapter])

	defer func() {
		testutils.ErrorIs(t, logger.Dispose(), ErrAdapter)
	}()

	record := slog.Record{
		Level:   LevelInfo.Level(),
		Message: "test message",
		Time:    time.Now(),
	}

	err := bufadapter.Handle(context.Background(), record)
	testutils.NoError(t, err, "Handle() should not immediately error") // Error happens in background

	// Give time for background processing
	time.Sleep(time.Millisecond * 100)

	adapterErr := bufadapter.Err()
	testutils.Error(t, adapterErr, "Adapter should have error after background processing")

	testutils.ErrorIs(t, adapterErr, ErrWriter, "Adapter should have logging.ErrWriter error")
	testutils.ErrorIs(t, adapterErr, ErrWriterIO, "Adapter should have logging.ErrWriterIO error")
	testutils.ErrorIs(t, adapterErr, ErrAdapter, "Adapter should have logging.ErrAdapter error")
}

func TestBufferedAdapterInitialization(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "DefaultConfig",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize: 1024,
					Policy:     AdapterPolicyBlock,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
		},
		{
			name: "ZeroBufferSizeBlock",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize: 0,
					Policy:     AdapterPolicyBlock,
				},
				SetSlogOutput: false,
			},
		},
		{
			name: "ZeroBufferSizeDrop",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize: 0,
					Policy:     AdapterPolicyDrop,
				},
				SetSlogOutput: false,
			},
		},
		{
			name: "ZeroBufferSizeCustomPolicy",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize: 0,
					Policy:     AdapterPolicy(100),
				},
				SetSlogOutput: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := NewBuffer()
			adapterI := NewBufferedTextAdapter(buf, nil).Compose(tt.config)
			logger := New(tt.config, adapterI)
			defer func() {
				testutils.NoError(t, logger.Dispose())
			}()

			testutils.NotNil(t, logger, "Logger should be initialized")
			logger.Enabled(context.Background(), LevelInfo.Level())

			adapter := adapterI.(*adapter)
			bufadapter := adapter.handler.(*BufferedAdapter[Adapter])

			actualBuffSize := bufadapter.records.Cap()
			expectedBufSize := tt.config.Adapter.BufferSize
			if expectedBufSize == 0 {
				expectedBufSize = DefaultAdapterBufferSize
			}
			testutils.Assert(t, expectedBufSize == actualBuffSize, "Expected buffer size %d got %d", expectedBufSize, actualBuffSize)
		})
	}
}

func TestNewBufferedAdapter(t *testing.T) {
	config := defaultTestConfig()
	adapter := NewAdapterWithHandler(&testWriter{fail: true}, slog.NewTextHandler).Compose(config)
	config.Adapter.BufferSize = 15
	badapter := NewBufferedAdapter(adapter, config.Adapter, nil)
	testutils.NotNil(t,
		badapter.WithGroup("group"),
		"badapter.WithAttrs should return with group",
	)
	testutils.ErrorIs(t, badapter.Dispose(), errLoggingTest)
	logger := New(config, badapter)
	testutils.NotNil(t,
		badapter.WithAttrs([]slog.Attr{slog.String("key", "value")}),
		"badapter.WithAttrs should return self, even if adapter is disposed",
	)
	testutils.NotNil(t,
		logger.With(slog.String("key", "value")),
		"logger.With should return logger, even if adapter is disposed",
	)
	testutils.NotNil(t,
		badapter.WithGroup("group"),
		"badapter.WithAttrs should return self, even if adapter is disposed",
	)
	testutils.NotNil(t,
		logger.WithGroup("group"),
		"logger.With should return logger, even if adapter is disposed",
	)
	testutils.Error(t,
		badapter.Handle(context.Background(), slog.Record{
			Message: "string",
		}),
		"badapter.Handle should return error when adapter is disposed",
	)
	testutils.Error(t,
		badapter.Flush(),
		"badapter.Flush should return error when adapter is disposed",
	)
	testutils.Assert(t,
		!badapter.Enabled(context.Background(), slog.LevelError),
		"badapter.Enabled should return false when adapter is disposed",
	)
}

func TestBufferedBuiltinAdapterDispose(t *testing.T) {
	buf := NewBuffer()
	textad := NewBufferedTextAdapter(buf, nil).Compose(defaultTestConfig()).(DisposableAdapter)
	jsonad := NewBufferedJSONAdapter(buf, nil).Compose(defaultTestConfig()).(DisposableAdapter)
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

func TestBufferedAdapterDispose(t *testing.T) {
	logger, buf := newBufferedLogger(defaultTestConfig(), nil)
	// Log a record
	err := logger.Handler().Handle(context.Background(), slog.Record{
		Level:   LevelInfo.Level(),
		Message: "logged-message",
	})
	testutils.NoError(t, err, "Handle() unexpected error")

	// Dispose and verify
	err = logger.Dispose()
	testutils.NoError(t, err, "Dispose() unexpected error")

	err = logger.Handler().Handle(context.Background(), slog.Record{
		Level:   LevelInfo.Level(),
		Message: "post-dispose message",
	})
	testutils.Equal(t, 128, buf.Cap(), "unexpected buffer capacity")
	testutils.Equal(t, 70, buf.Len(), "unexpected buffer length")
	str := buf.String()
	testutils.NoError(t, err, "Handle() after Dispose should not error")
	if str != "level=info msg=logged-message\n{\"level\":\"info\",\"msg\":\"logged-message\"}\n" {
		testutils.Equal(t, "{\"level\":\"info\",\"msg\":\"logged-message\"}\nlevel=info msg=logged-message\n", str)
	}
	testutils.Equal(t, 2, strings.Count(str, "\n"), "Dispose() should process pending records")

	var b []byte
	n, err := buf.Read(b)
	testutils.NoError(t, err)
	testutils.Equal(t, 0, n, "unexpected buffer read length")
}

func TestBufferedAdapterBufferFullDrop(t *testing.T) {
	t.Skip()
	config := defaultTestConfig()
	config.Adapter.BufferSize = 10
	config.Adapter.Policy = AdapterPolicyDrop
	config.Adapter.FlushInterval = time.Second
	config.seal()

	droppedRecords := expvar.NewInt(fmt.Sprintf("logging.buffered_adapter.dropped_records_%s", t.Name()))
	adapterI := NewBufferedTextAdapter(NewBuffer(), droppedRecords).Compose(config)
	logger := New(config, adapterI)
	defer func() {
		testutils.NoError(t, logger.Dispose())
	}()

	// Reset droppedLogs for test isolation
	droppedRecords.Set(0)
	ctx := context.Background()

	// Give queue time to process initial setup
	time.Sleep(time.Millisecond * 10)

	adapter := adapterI.(*adapter)
	bufadapter := adapter.handler.(*BufferedAdapter[Adapter])

	adapter.w.mu.Lock()
	defer adapter.w.mu.Unlock()

	expectedCapacity := int(config.Adapter.BufferSize) // Channel capacity

	var firstError error
	successCount := 0
	errorCount := 0

	for i := 0; i < expectedCapacity+10; i++ {
		err := bufadapter.Handle(ctx, slog.Record{
			Level:   LevelInfo.Level(),
			Message: fmt.Sprintf("message %d", i),
		})

		if err == nil {
			successCount++
		} else {
			errorCount++
			if firstError == nil {
				firstError = err
			}
			testutils.Assert(t,
				errors.Is(err, ErrAdapterBufferFull),
				"Expected ErrAdapterBufferFull, got: %v", err)
		}
	}

	// Verify that we could buffer exactly the expected capacity
	testutils.Assert(t, successCount == expectedCapacity,
		"Expected to buffer %d items successfully, got %d", expectedCapacity, successCount)

	// Verify that subsequent items were dropped
	testutils.Assert(t, errorCount > 0,
		"Expected some items to be dropped, but got %d errors", errorCount)

	// Verify the first error was ErrAdapterBufferFull
	testutils.Assert(t, firstError != nil && errors.Is(firstError, ErrAdapterBufferFull),
		"Expected ErrAdapterBufferFull as first error")
}

func TestBufferedAdapterFlush(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		recordsCount  int
		expectedLines int
		expectDropped bool
		expectError   bool
	}{

		{
			name: "BlockPolicySmallCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    4,
					Policy:        AdapterPolicyBlock,
					FlushInterval: 2,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  2,
			expectedLines: 4,
			expectDropped: false,
		},
		{
			name: "BlockPolicyMatchCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    2,
					Policy:        AdapterPolicyBlock,
					FlushInterval: 0,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  2,
			expectedLines: 4,
			expectDropped: false,
		},
		{
			name: "BlockPolicyAboveBufferSize",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    4,
					Policy:        AdapterPolicyBlock,
					FlushInterval: 0,
					MaxRetries:    5,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  32,
			expectedLines: 64,
			expectDropped: false,
		},
		{
			name: "BlockPolicyHighCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    4,
					Policy:        AdapterPolicyBlock,
					FlushInterval: time.Second,
					MaxRetries:    1,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  1024,
			expectedLines: 8,
			expectDropped: true,
			expectError:   true,
		},
		{
			name: "DropPolicyMatchCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    2,
					Policy:        AdapterPolicyDrop,
					FlushInterval: 0,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  2,
			expectedLines: 4,
			expectDropped: false,
		},
		{
			name: "DropPolicySmallCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    4,
					Policy:        AdapterPolicyDrop,
					FlushInterval: 0,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  2,
			expectedLines: 4,
			expectDropped: false,
		},
		{
			name: "DropPolicyHighCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    2,
					Policy:        AdapterPolicyDrop,
					FlushInterval: 0,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  32,
			expectedLines: 4,
			expectDropped: true,
			expectError:   false,
		},

		{
			name: "CustomPolicyMatchCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    2,
					Policy:        AdapterPolicy(100),
					FlushInterval: 0,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  2,
			expectedLines: 4,
			expectDropped: false,
		},
		{
			name: "CustomPolicySmallCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    4,
					Policy:        AdapterPolicy(100),
					FlushInterval: time.Second,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
					MaxRetries:    DefaultAdapterMaxRetries,
					RetryTimeout:  DefaultAdapterRetryTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  2,
			expectedLines: 4,
			expectDropped: false,
		},
		{
			name: "CustomPolicyHighCount",
			config: Config{
				Level: LevelInfo,
				Adapter: AdapterConfig{
					BufferSize:    4,
					Policy:        AdapterPolicy(100),
					FlushInterval: time.Second,
					MaxRetries:    10,
					RetryTimeout:  time.Millisecond * 100,
					BatchSize:     DefaultAdapterBatchSize,
					FlushTimeout:  DefaultAdapterFlushTimeout,
				},
				TimeFormat:    "15:04:05.000",
				TimeLocation:  time.UTC,
				SetSlogOutput: false,
			},
			recordsCount:  128,
			expectedLines: 256,
			expectDropped: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			droppedRecords := expvar.NewInt(fmt.Sprintf("logging.buffered_adapter.dropped_records_%s", t.Name()))
			logger, buf := newBufferedLogger(tt.config, droppedRecords)
			defer func() {
				_ = logger.Dispose()
			}()

			testutils.NotNil(t, logger, "Logger should never be nil")

			// Reset droppedLogs for test isolation
			droppedRecords.Set(0)

			// Log multiple records

			var (
				recordsCount     int
				expectDroppedErr bool
			)
			for i := range tt.recordsCount {

				err := logger.Handler().Handle(t.Context(), slog.Record{
					Level:   LevelInfo.Level(),
					Message: "test message " + string(rune('A'+i)),
					Time:    time.Date(2025, 9, 6, 12, 0, 0, 0, time.UTC),
				})
				testutils.NoError(t, logger.Handler().Handle(t.Context(), NewHttpRecord(time.Now(), http.MethodConnect, 0, "/")))

				if tt.config.Adapter.Policy == AdapterPolicyDrop && i >= tt.config.Adapter.BufferSize {
					if tt.expectDropped {
						expectDroppedErr = true
						testutils.Error(t, err, "Handle() expected error for dropped record")
						continue
					}
				} else if tt.config.Adapter.Policy != AdapterPolicyDrop &&
					i >= tt.config.Adapter.BufferSize && tt.expectError {
					expectDroppedErr = true
					testutils.Error(t, err, "Handle() expected error for exhausted buffer")
				} else if tt.config.Adapter.Policy == AdapterPolicyBlock && errors.Is(err, ErrAdapterBufferFull) {
					expectDroppedErr = true
				}

				if !expectDroppedErr {
					if testutils.NoError(t, err,
						"Handle() unexpected error i=%d buffer_size=%d Drop=%t",
						i,
						tt.config.Adapter.BufferSize,
						(tt.config.Adapter.Policy == AdapterPolicyDrop),
					) {
						recordsCount++
					} else {
						t.FailNow()
					}
					continue
				}
			}

			flushErr := logger.Flush()

			if tt.expectDropped {
				testutils.Assert(t, droppedRecords.Value() > 0, "should have dropped records")
			} else if tt.expectError {
				testutils.Error(t, flushErr)
			} else {
				testutils.NoError(t, flushErr)
			}

			str := buf.String()
			testutils.Assert(t, len(str) > 0, "expected reader to read more than 0 bytes")
			testutils.Equal(t, tt.expectedLines, strings.Count(str, "\n"), "Flush() should process expected number of records")
		})
	}
}
