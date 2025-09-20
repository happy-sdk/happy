// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package adapters

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/tui/ansicolor"
	"golang.org/x/sync/errgroup"
)

func TestConsoleAdapterCustomTheme(t *testing.T) {
	config := logging.DefaultConfig()
	config.Level = logging.LevelInfo
	config.SetSlogOutput = false

	// Create custom theme with different colors
	customTheme := ConsoleAdapterTheme{
		TimeLabel:  ansicolor.Style{FG: ansicolor.RGB(255, 0, 0)},   // Red
		SourceLink: ansicolor.Style{FG: ansicolor.RGB(0, 255, 0)},   // Green
		Attrs:      ansicolor.Style{FG: ansicolor.RGB(0, 0, 255)},   // Blue
		LevelInfo:  ansicolor.Style{FG: ansicolor.RGB(255, 255, 0)}, // Yellow
	}

	buf := logging.NewBuffer()
	logger := logging.New(config, NewConsoleAdapter(buf, customTheme))
	adapter := logging.GetAdaptersFromHandler[*ConsoleAdapter](logger.Handler())[0]
	defer func() {
		if err := logger.Dispose(); err != nil {
			t.Fatal(err)
		}
	}()

	record := slog.Record{
		Level:   logging.LevelInfo.Level(),
		Message: "custom theme test",
		Time:    time.Date(2025, 9, 6, 12, 0, 0, 0, time.UTC),
	}
	record.AddAttrs(slog.String("key", "value"))

	err := adapter.Handle(context.Background(), record)
	testutils.NoError(t, err, "Handle() unexpected error")

	output := buf.String()
	testutils.Assert(t, strings.Contains(output, "custom theme test"), "Output should contain the message")
	testutils.Assert(t, strings.Contains(output, "\x1b[38;2;255;255;0m"), "Output should contain custom yellow color for info level")
	testutils.Assert(t, strings.Contains(output, "\x1b[38;2;255;0;0m"), "Output should contain custom red color for timestamp")
	testutils.Assert(t, strings.Contains(output, "\x1b[38;2;0;0;255m"), "Output should contain custom blue color for attributes")
}

func TestConsoleAdapterThemeUnknownLevel(t *testing.T) {
	theme := ConsoleAdapterDefaultTheme().Build()

	// Test with a custom level not in the theme
	customLevel := logging.Level(999)
	levelStr := theme.LevelString(customLevel)

	expected := fmt.Sprintf(" %-12s", customLevel.String())
	testutils.Equal(t, expected, levelStr, "Unknown level should be formatted with default padding")
}
func TestConsoleAdapterBatchAndErrorHandling(t *testing.T) {
	config := logging.DefaultConfig()
	config.Level = logging.LevelDebug
	config.TimeFormat = "2006-01-02 15:04:05"
	config.SetSlogOutput = false
	config.AddSource = true

	// Custom theme with distinct colors for verification
	customTheme := ConsoleAdapterTheme{
		TimeLabel:  ansicolor.Style{FG: ansicolor.RGB(255, 128, 128)}, // Light red
		SourceLink: ansicolor.Style{FG: ansicolor.RGB(128, 255, 128)}, // Light green
		Attrs:      ansicolor.Style{FG: ansicolor.RGB(128, 128, 255)}, // Light blue
		LevelDebug: ansicolor.Style{FG: ansicolor.RGB(255, 255, 128)}, // Light yellow
		LevelInfo:  ansicolor.Style{FG: ansicolor.RGB(128, 255, 255)}, // Cyan
		LevelError: ansicolor.Style{FG: ansicolor.RGB(255, 0, 0)},     // Red
	}

	// Use sync.Pool for buffer to simulate production usage
	var bufPool = sync.Pool{New: func() any { return bytes.NewBuffer(make([]byte, 0, 1024)) }}
	getBuf := func() *bytes.Buffer { return bufPool.Get().(*bytes.Buffer) }
	putBuf := func(b *bytes.Buffer) { b.Reset(); bufPool.Put(b) }

	t.Run("BatchHandlingWithSource", func(t *testing.T) {
		buf := getBuf()
		defer putBuf(buf)

		logger := logging.New(config, NewConsoleAdapter(buf, customTheme))
		adapter := logging.GetAdaptersFromHandler[*ConsoleAdapter](logger.Handler())[0]
		defer func() {
			if err := logger.Dispose(); err != nil {
				t.Fatal(err)
			}
		}()

		// Create multiple records with source info
		records := []logging.Record{
			{
				Record: slog.Record{
					Level:   logging.LevelDebug.Level(),
					Message: "debug message",
					Time:    time.Date(2025, 9, 8, 10, 0, 0, 0, time.UTC),
				},
			},
			{
				Record: slog.Record{
					Level:   logging.LevelInfo.Level(),
					Message: "info message",
					Time:    time.Date(2025, 9, 8, 10, 0, 1, 0, time.UTC),
				},
			},
			{
				Record: slog.Record{
					Level:   logging.LevelError.Level(),
					Message: "error message",
					Time:    time.Date(2025, 9, 8, 10, 0, 2, 0, time.UTC),
				},
			},
		}

		// Add source info to second record
		records[1].Record.AddAttrs(slog.Any("source", &slog.Source{
			File:     "/path/to/file.go",
			Line:     42,
			Function: "main.TestFunction",
		}))
		// Add attributes to third record
		records[2].Record.AddAttrs(slog.String("error_code", "E123"))

		// Test batch handling
		err := adapter.BatchHandle(records)
		testutils.NoError(t, err, "BatchHandler() unexpected error")

		output := buf.String()
		// Verify level colors
		testutils.Assert(t, strings.Contains(output, "\x1b[38;2;255;255;128m"), "Output should contain debug level color")
		testutils.Assert(t, strings.Contains(output, "\x1b[38;2;128;255;255m"), "Output should contain info level color")
		testutils.Assert(t, strings.Contains(output, "\x1b[38;2;255;0;0m"), "Output should contain error level color")

		// Verify messages
		testutils.Assert(t, strings.Contains(output, "debug message"), "Output should contain debug message")
		testutils.Assert(t, strings.Contains(output, "info message"), "Output should contain info message")
		testutils.Assert(t, strings.Contains(output, "error message"), "Output should contain error message")

		// Verify timestamp and source
		testutils.Assert(t, strings.Contains(output, "2025-09-08 10:00:00"), "Output should contain timestamp")
		testutils.Assert(t, strings.Contains(output, "\x1b[38;2;128;255;128m/path/to/file.go:42"), "Output should contain source link")
		testutils.Assert(t, strings.Contains(output, "TestFunction"), "Output should contain function name")

		// Verify attributes
		testutils.Assert(t, strings.Contains(output, "\x1b[38;2;128;128;255m{\"error_code\":\"E123\"}"), "Output should contain attributes")
	})

	t.Run("ErrorPropagation", func(t *testing.T) {

		fw := &failingWriter{
			shouldFail: true,
		}
		logger := logging.New(config, NewConsoleAdapter(fw, customTheme))
		adapter := logging.GetAdaptersFromHandler[*ConsoleAdapter](logger.Handler())[0]
		defer func() {
			if err := logger.Dispose(); err != nil {
				t.Fatal(err)
			}
		}()

		record := slog.Record{
			Level:   logging.LevelInfo.Level(),
			Message: "test error",
			Time:    time.Now(),
		}

		err := adapter.Handle(context.Background(), record)
		testutils.Error(t, err, "Handle() should return error for failing writer")
		testutils.Assert(t, strings.Contains(err.Error(), "write failed"), "Error should contain write failure message")

	})

	t.Run("ConcurrentBatchHandling", func(t *testing.T) {
		buf := getBuf()
		defer putBuf(buf)

		logger := logging.New(config, NewConsoleAdapter(buf, customTheme))
		adapter := logging.GetAdaptersFromHandler[*ConsoleAdapter](logger.Handler())[0]
		defer func() {
			if err := logger.Dispose(); err != nil {
				t.Fatal(err)
			}
		}()

		var g errgroup.Group
		const numGoroutines = 5
		const recordsPerGoroutine = 10

		for i := range numGoroutines {
			g.Go(func() error {
				records := make([]logging.Record, recordsPerGoroutine)
				for j := range recordsPerGoroutine {
					records[j] = logging.Record{
						Record: slog.Record{
							Level:   logging.LevelDebug.Level(),
							Message: fmt.Sprintf("message-%d-%d", i, j),
							Time:    time.Date(2025, 9, 8, 10, 0, 0, 0, time.UTC),
						},
					}
				}
				return adapter.BatchHandle(records)
			})
		}

		err := g.Wait()
		testutils.NoError(t, err, "Concurrent BatchHandler() unexpected error")

		output := buf.String()
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < recordsPerGoroutine; j++ {
				testutils.Assert(t, strings.Contains(output, fmt.Sprintf("message-%d-%d", i, j)), "Output should contain all messages")
			}
		}
	})
}

// Helper function to print hex representation for debugging
// func printHex(t *testing.T, name, str string) {
// 	t.Logf("%s hex: %x", name, ansicolor.StringHEX(str))
// 	t.Logf("%s string: %q", name, str)
// 	t.Logf("%s stripped: %q", name, ansicolor.StrippedString(str))
// }
