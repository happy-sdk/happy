// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors
//
// Internal benchmark suite comparing Happy logger with std log, slog
// and a few well-known third-party loggers.
//
// This lives in its own internal module so third-party dependencies
// never leak into the main Happy-SDK module or its transitive closure.

package bench

import (
	"bytes"
	"context"
	"io"
	"log"
	"log/slog"
	"testing"

	hlogging "github.com/happy-sdk/happy/pkg/logging"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// common benchmark payload: a realistic structured log line.
var (
	msg       = "user logged in"
	fieldsAny = []any{
		"user_id", 12345,
		"role", "admin",
		"success", true,
		"latency_ms", 12.34,
	}
)

// newHappyLogger builds a Happy logger configured for benchmark use.
func newHappyLogger(w io.Writer) *hlogging.Logger {
	cfg := hlogging.DefaultConfig()
	cfg.SetSlogOutput = false
	cfg.Level = hlogging.LevelInfo

	adapter := hlogging.NewTextAdapter(w)
	return hlogging.New(cfg, adapter)
}

// newStdLog builds a std log.Logger writing to w.
func newStdLog(w io.Writer) *log.Logger {
	// Keep flags minimal to approximate Happy default output cost.
	return log.New(w, "", log.LstdFlags)
}

// newSlog builds a slog.Logger writing to w using options that roughly match
// the Happy logger's default text adapter.
func newSlog(w io.Writer) *slog.Logger {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler)
}

// newZapLogger builds a zap.Logger with a text encoder writing to w.
func newZapLogger(w io.Writer) *zap.Logger {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encCfg),
		zapcore.AddSync(w),
		zapcore.InfoLevel,
	)
	return zap.New(core)
}

// newLogrusLogger builds a logrus.Logger writing to w.
func newLogrusLogger(w io.Writer) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(w)
	l.SetLevel(logrus.InfoLevel)
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
	})
	return l
}

// newZerolog builds a zerolog.Logger writing to w.
func newZerolog(w io.Writer) zerolog.Logger {
	// human-friendly console writer, closest to default Happy text adapter
	console := zerolog.New(w).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Logger()
	return console
}

// BenchmarkCommonUseCase benchmarks a single, common structured logging
// pattern across different logging implementations.
//
// The goal is not to pick a winner, but to detect regressions in our
// logger relative to other options and over time.
func BenchmarkCommonUseCase(b *testing.B) {
	b.Run("stdlog", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		l := newStdLog(w)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.Printf("%s user_id=%d role=%s success=%t latency_ms=%.2f",
				msg, 12345, "admin", true, 12.34)
		}
	})

	b.Run("slog(slog-attrs)", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		l := newSlog(w)
		ctx := contextWithNow()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.InfoContext(ctx, msg,
				slog.Int("user_id", 12345),
				slog.String("role", "admin"),
				slog.Bool("success", true),
				slog.Float64("latency_ms", 12.34),
			)
		}
	})

	b.Run("slog(fields-any)", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		l := newSlog(w)
		ctx := contextWithNow()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l.InfoContext(ctx, msg, fieldsAny...)
		}
	})

	b.Run("happy-logger(fields-any)", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		logger := newHappyLogger(w)
		defer func() { _ = logger.Dispose() }()

		ctx := contextWithNow()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.InfoContext(ctx, msg, fieldsAny...)
		}
	})

	b.Run("happy-logger(slog-attrs)", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		logger := newHappyLogger(w)
		defer func() { _ = logger.Dispose() }()

		ctx := contextWithNow()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.InfoContext(ctx, msg,
				slog.Int("user_id", 12345),
				slog.String("role", "admin"),
				slog.Bool("success", true),
				slog.Float64("latency_ms", 12.34),
			)
		}
	})

	b.Run("zap", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		logger := newZapLogger(w)
		defer func() { _ = logger.Sync() }()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info(msg,
				zap.Int("user_id", 12345),
				zap.String("role", "admin"),
				zap.Bool("success", true),
				zap.Float64("latency_ms", 12.34),
			)
		}
	})

	b.Run("logrus", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		logger := newLogrusLogger(w)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.WithFields(logrus.Fields{
				"user_id":    12345,
				"role":       "admin",
				"success":    true,
				"latency_ms": 12.34,
			}).Info(msg)
		}
	})

	b.Run("zerolog", func(b *testing.B) {
		w := newSink()
		defer w.Close()

		logger := newZerolog(w)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info().
				Int("user_id", 12345).
				Str("role", "admin").
				Bool("success", true).
				Float64("latency_ms", 12.34).
				Msg(msg)
		}
	})
}

// contextWithNow returns a context that could be enriched with values
// if needed; for now it just provides a consistent place to hang
// deadline/cancellation in future benchmarks.
func contextWithNow() context.Context {
	// In the future we might attach trace/span IDs here for tracing benchmarks.
	return context.Background()
}

// newSink returns an io.WriteCloser suitable for benchmarks.
// Using a per-benchmark buffer avoids fd ownership issues between loggers
// (some may Close their writers) while still exercising encoding and
// formatting cost.
func newSink() io.WriteCloser {
	return nopCloser{Writer: &bytes.Buffer{}}
}

type nopCloser struct {
	io.Writer
}

func (n nopCloser) Close() error { return nil }
