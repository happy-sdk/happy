// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"bytes"
	"fmt"
	"log/slog"
	"testing"
	"time"
)

func BenchmarkLoggers(b *testing.B) {
	args := []any{
		"key1", "value1",
		"key2", 42,
		"key3", true,
		"key4", "value4",
	}

	config := DefaultConfig()
	config.SetSlogOutput = false

	slogLogger := slog.New(slog.NewTextHandler(NewBuffer(), config.HandlerOptions()))

	b.Run("Slog", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			slogLogger.Log(b.Context(), slog.LevelInfo, "log info message", args...)
		}
	})

	logger := New(
		config,
		NewTextAdapter(NewBuffer()),
	)
	b.Run("Logger", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			logger.Log(b.Context(), slog.LevelInfo, "log info message", args...)
		}
		_ = logger.Flush()
	})

	tests := []struct {
		Name                 string
		BufferSize           int
		AdapterPolicy        AdapterPolicy
		AdapterBatchSize     int
		AdapterFlushInterval time.Duration
		AdapterMaxRetries    int
		AdapterRetryTimeout  time.Duration
	}{
		{
			Name:                 "Buffered_Drop_b128_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyDrop,
			AdapterBatchSize:     128,
			AdapterFlushInterval: DefaultAdapterFlushInterval, // 0 immediate
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Drop_i1024_b128_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyDrop,
			AdapterBatchSize:     128,
			AdapterFlushInterval: time.Duration(time.Microsecond * 1024),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Drop_i1024_b128_s2048",
			BufferSize:           2048,
			AdapterPolicy:        AdapterPolicyDrop,
			AdapterBatchSize:     128,
			AdapterFlushInterval: time.Duration(time.Microsecond * 1024),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Drop_i1024_b128_s4096",
			BufferSize:           4096,
			AdapterPolicy:        AdapterPolicyDrop,
			AdapterBatchSize:     128,
			AdapterFlushInterval: time.Duration(time.Microsecond * 1024),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Drop_i1024_b128_s8192",
			BufferSize:           8192,
			AdapterPolicy:        AdapterPolicyDrop,
			AdapterBatchSize:     128,
			AdapterFlushInterval: time.Duration(time.Microsecond * 1024),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},

		{
			Name:                 "Buffered_Block_0_b128_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     128,
			AdapterFlushInterval: 0, // 0 immediate
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},

		{
			Name:                 "Buffered_Block_i128_b128_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     128,
			AdapterFlushInterval: time.Duration(time.Microsecond * 128),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},

		{
			Name:                 "Buffered_Block_i256_b128_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     128,
			AdapterFlushInterval: time.Duration(time.Microsecond * 256),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},

		{
			Name:                 "Buffered_Block_i1024_b128_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     128,
			AdapterFlushInterval: time.Duration(time.Microsecond * 1024),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},

		// Drops

		{
			Name:                 "Buffered_Block_i128_b256_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     256,
			AdapterFlushInterval: time.Duration(time.Microsecond * 128),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i256_b256_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     256,
			AdapterFlushInterval: time.Duration(time.Microsecond * 256),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},

		{
			Name:                 "Buffered_Block_i512_b256_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     256,
			AdapterFlushInterval: time.Duration(time.Microsecond * 512),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i512_b1024_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     1024,
			AdapterFlushInterval: time.Duration(time.Microsecond * 512),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},

		{
			Name:                 "Buffered_Block_i512_b4096_s1024",
			BufferSize:           1024,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     4096,
			AdapterFlushInterval: time.Duration(time.Microsecond * 512),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i512_b1024_s2048",
			BufferSize:           2048,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     1024,
			AdapterFlushInterval: time.Duration(time.Microsecond * 512),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i512_b1024_s4096",
			BufferSize:           4096,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     1024,
			AdapterFlushInterval: time.Duration(time.Microsecond * 512),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i512_b1024_s8192",
			BufferSize:           8192,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     1024,
			AdapterFlushInterval: time.Duration(time.Microsecond * 512),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i1ms_b512_s8192",
			BufferSize:           8192,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     512,
			AdapterFlushInterval: time.Duration(time.Millisecond),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i1ms_b1024_s8192",
			BufferSize:           8192,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     1024,
			AdapterFlushInterval: time.Duration(time.Millisecond),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i1ms_b2048_s8192",
			BufferSize:           8192,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     2048,
			AdapterFlushInterval: time.Duration(time.Millisecond),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i512µs_b2048_s8192",
			BufferSize:           8192,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     2048,
			AdapterFlushInterval: time.Duration(time.Microsecond * 512),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i256µs_b2048_s8192",
			BufferSize:           8192,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     2048,
			AdapterFlushInterval: time.Duration(time.Microsecond * 256),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i256µs_b2048_s16384",
			BufferSize:           16384,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     2048,
			AdapterFlushInterval: time.Duration(time.Microsecond * 256),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_i256µs_b4096_s16384",
			BufferSize:           16384,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     4096,
			AdapterFlushInterval: time.Duration(time.Microsecond * 256),
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
		{
			Name:                 "Buffered_Block_Defaults",
			BufferSize:           DefaultAdapterBufferSize,
			AdapterPolicy:        AdapterPolicyBlock,
			AdapterBatchSize:     DefaultAdapterBatchSize,
			AdapterFlushInterval: DefaultAdapterFlushInterval,
			AdapterMaxRetries:    DefaultAdapterMaxRetries,
			AdapterRetryTimeout:  DefaultAdapterRetryTimeout,
		},
	}

	for _, tt := range tests {
		config := DefaultConfig()
		config.Adapter.BufferSize = tt.BufferSize
		config.Adapter.Policy = tt.AdapterPolicy
		config.Adapter.BatchSize = tt.AdapterBatchSize
		config.Adapter.FlushInterval = tt.AdapterFlushInterval
		config.Adapter.MaxRetries = tt.AdapterMaxRetries
		config.Adapter.RetryTimeout = tt.AdapterRetryTimeout
		config.AttrProcessorPoolSize = 4
		config.SetSlogOutput = false
		logger := New(
			config,
			NewBufferedTextAdapter(NewBuffer(), nil),
		)

		b.Run(tt.Name, func(b *testing.B) {
			defer func() {
				_ = logger.Dispose()
			}()
			b.ReportAllocs()
			for b.Loop() {
				logger.Log(b.Context(), slog.LevelInfo, "log info message", args...)
			}
		})
	}

	// b.Run("Buffered_Drop_i1024_s8192_Concurrent", func(b *testing.B) {
	// 	config := DefaultConfig()
	// 	config.Adapter.BufferSize = 8192
	// 	config.Adapter.Policy = AdapterPolicyDrop
	// 	config.Adapter.FlushInterval = time.Microsecond * 1024
	// 	config.Adapter.BatchSize = 1024
	// 	logger := New(config, NewBufferedTextAdapter(NewBuffer(), nil))
	// 	defer logger.Dispose()
	// 	b.ReportAllocs()
	// 	b.ResetTimer()
	// 	b.RunParallel(func(pb *testing.PB) {
	// 		for pb.Next() {
	// 			logger.Log(b.Context(), slog.LevelInfo, "log info message", args...)
	// 		}
	// 	})
	// })
	// b.Run("Buffered_Block_i1024_s8192_Concurrent", func(b *testing.B) {
	// 	config := DefaultConfig()
	// 	config.Adapter.BufferSize = 8192
	// 	config.Adapter.Policy = AdapterPolicyBlock
	// 	config.Adapter.FlushInterval = time.Microsecond * 1024
	// 	config.Adapter.BatchSize = 1024
	// 	logger := New(config, NewBufferedTextAdapter(NewBuffer(), nil))
	// 	defer logger.Dispose()
	// 	b.ReportAllocs()
	// 	b.ResetTimer()
	// 	b.RunParallel(func(pb *testing.PB) {
	// 		for pb.Next() {
	// 			logger.Log(b.Context(), slog.LevelInfo, "log info message", args...)
	// 		}
	// 	})
	// })
}

type lineWriter interface {
	Write(p []byte) (n int, err error)
	WriteByte(c byte) error
	WriteString(s string) (n int, err error)
	String() string
}

func buildLogLine[W lineWriter](buf W, attrs []slog.Attr) string {
	for i, attr := range attrs {
		if i > 0 {
			_ = buf.WriteByte(' ')
		}
		_, _ = buf.WriteString(attr.Key)
		_ = buf.WriteByte('=')
		_, _ = fmt.Fprint(buf, attr.Value.Any())
	}
	return buf.String()
}

func BenchmarkLineBuffer(b *testing.B) {
	attrs := []slog.Attr{
		slog.String("key1", "value1"),
		slog.Int("key2", 42),
		slog.Bool("key3", true),
		slog.String("key4", "value4"),
	}

	b.Run("LineBuffer_Pooled", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			buf := NewLineBuffer()
			_ = buildLogLine(buf, attrs)
			buf.Free()
		}
	})

	b.Run("BytesBuffer_New", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var buf bytes.Buffer
			_ = buildLogLine(&buf, attrs)
		}
	})

	b.Run("BytesBuffer_Preallocated", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			buf := bytes.Buffer{}
			buf.Grow(1024) // Match LineBuffer's initial capacity
			_ = buildLogLine(&buf, attrs)
		}
	})
}
