// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

// Package logging provides a high-performance, flexible logging framework compatible
// with log/slog, designed for modern Go applications. It supports multiple adapters
// with distinct behaviors (e.g., buffered with drop for metrics, block for reliable
// logging, or synchronous for console output), enabling tailored log handling.
//
// Features:
//   - Slog Compatibility: Implements slog.Handler, supporting all standard methods
//     (Log, Debug, Info, With, etc.).
//   - Multi-Adapter Support: Assign multiple adapters (e.g., BufferedTextAdapter,
//     BufferedJSONAdapter, ConsoleAdapter) to log to different outputs with custom
//     behaviors (e.g., Drop for metrics, Block for audits). Buffered logs are processed
//     in batches for efficiency.
//   - Hot-Swappable Writers: Swap adapter writers at runtime (e.g., from os.Stdout to
//     a file for log rotation or daemonizing) without recreating adapters or affecting
//     other adapters. Utility functions support swapping os.Stdout/os.Stderr for all
//     relevant adapters, leaving buffered or custom adapters unchanged.
//   - BufferedAdapter: Offers high throughput (~3x faster than slog.Logger difault handlerers
//     with Drop policy, ~10x in concurrent workloads) or reliable logging (~24% faster with Block
//     policy).
//   - Configurability: AdapterConfig tunes BufferSize (default 8192, ~5-10 MiB),
//     BatchSize (default 2048), FlushInterval (default 512µs), and Policy (default Block).
//   - Resource Management: Dispose and Flush ensure proper cleanup and log syncing.
//     Dispose MUST be called when application exits on primary logger to signal all adapters
//     gracefully release resources.
//   - Derived Loggers: WithGroup/With create lightweight loggers sharing state, no
//     separate Dispose needed.
//
// Usage:
//
//	config := DefaultConfig()
//	config.Level = LevelInfo
//	logger := New(config,
//	    NewBufferedTextAdapter(os.Stdout, nil), // Block policy for reliable logs
//	    NewBufferedJSONAdapter(os.Stderr, &AdapterConfig{Policy: AdapterPolicyDrop}), // Drop for metrics
//	    adapters.NewConsoleAdapter(os.Stdout, adapters.ConsoleAdapterDefaultTheme()), // Synchronous
//	)
//	defer logger.Dispose()
//	logger.Info("message", "key", "value")
//	// Swap writer for BufferedTextAdapter (e.g., for log rotation)
//	logger.ReplaceWriter(0, os.Stdout, newLogFile)
//	logger.Flush()
//
// For memory-constrained environments, set BufferSize=2048 (~1.5 MiB) for minimal
// performance impact (~8% slower). See benchmarks in the repository for details.
package logging

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/happy-sdk/happy/pkg/bitutils"
)

// Constants for adapter configuration and policies.
const (
	// DefaultAdapterBufferSize is the default buffer size for buffered adapters (power of 2).
	DefaultAdapterBufferSize int = 8192 // records
	// DefaultAdapterBatchSize is the default batch size for buffered adapters.
	DefaultAdapterBatchSize int = 2048 // records
	// DefaultAdapterFlushInterval is the default flush interval (0 = immediate flush).
	DefaultAdapterFlushInterval time.Duration = time.Duration(time.Microsecond * 256)

	DefaultAdapterFlushTimeout time.Duration = time.Duration(time.Second * 5)
	// DefaultAdapterRetryTimeout is the max timeout for retries in AdapterPolicyBlock.
	DefaultAdapterRetryTimeout = time.Second
	// DefaultAdapterMaxRetries is the max retry attempts in AdapterPolicyBlock.
	DefaultAdapterMaxRetries = 10
	// DefaultAttrProcessorPoolSize is the default pool size for attribute processors.
	DefaultAttrProcessorPoolSize uint8 = 4
	// secretAttrValue is the placeholder for redacted secret attribute values.
	secretAttrValue = "<redacted>"
)

const (
	// AdapterPolicyBlock blocks the handler until the record is written or times out.
	AdapterPolicyBlock AdapterPolicy = iota
	// AdapterPolicyDrop discards records immediately when the buffer is full.
	AdapterPolicyDrop
)

var (
	// Error is the base error for the logging package.
	Error = errors.New("logging")
	// ErrLoggerDisposed is returned when entire logger and all Adapters are Disposed and Logger is used after disposal.
	ErrLoggerDisposed = fmt.Errorf("%w: logger disposed", Error)
	// ErrAdapter is the base error for adapter-related issues.
	ErrAdapter = fmt.Errorf("%w:adapter", Error)
	// ErrAdapterNotComposed is returned when an adapter is used before composition.
	ErrAdapterNotComposed = fmt.Errorf("%w: not composed", ErrAdapter)
	// ErrAdapterDisposed is returned when an adapter is used after disposal.
	ErrAdapterDisposed = fmt.Errorf("%w: disposed", ErrAdapter)
	// ErrAdapterSwappingOutput is returned when swapping adapter output fails.
	ErrAdapterSwappingOutput = fmt.Errorf("%w: swapping output", ErrAdapter)
	// ErrAdapterBufferFull is returned when a record is dropped due to a full buffer.
	ErrAdapterBufferFull = fmt.Errorf("%w: buffer full, record dropped", ErrAdapter)
	// ErrAdapterBufferFullRetry is returned when a buffer is full but a retry is needed.
	ErrAdapterBufferFullRetry = fmt.Errorf("%w: buffer full, should retry", ErrAdapter)
	// ErrWriter is the base error for writer-related issues.
	ErrWriter = fmt.Errorf("%w:writer", Error)
	// ErrWriterIO is returned when an I/O error occurs during a write operation.
	ErrWriterIO = fmt.Errorf("%w:io", ErrWriter)
	// ErrLevel is returned when a logging level is invalid or unsupported.
	ErrLevel = fmt.Errorf("%w:level", Error)

	// secretSlogAttrValue is the default value for redacted secret attributes.
	secretSlogAttrValue = slog.StringValue(secretAttrValue)
)

// AdapterPolicy defines the behavior for handling full buffers in buffered adapters.
type AdapterPolicy uint8

// String returns the string representation of the adapter policy.
func (p AdapterPolicy) String() string {
	switch p {
	case AdapterPolicyBlock:
		return "block"
	case AdapterPolicyDrop:
		return "drop"
	default:
		return "unknown"
	}
}

// ReplaceAttrFunc transforms a slog.Attr, optionally considering group context.
type ReplaceAttrFunc func(groups []string, a slog.Attr) slog.Attr

// Config defines settings for a logging handler,
// including level, attributes, and adapter behavior.
type Config struct {
	sealed                bool
	lvl                   *slog.LevelVar  // Variable log level.
	replaceAttr           ReplaceAttrFunc // Function to transform attributes.
	AddSource             bool            // Include source information in logs.
	AttrProcessorPoolSize uint8           // Size of attribute processor pool.
	Adapter               AdapterConfig   // Buffered adapter configuration.
	Level                 Level           // Minimum log level.
	NoTimestamp           bool            // Omit timestamps in logs.
	Secrets               []string        // Keys to redact as secrets.
	SetSlogOutput         bool            // Enable slog output integration.
	TimeFormat            string          // Timestamp format (empty if NoTimestamp).
	TimeLocation          *time.Location  // Timezone for timestamps.
	Omit                  []string        // Attribute keys to omit.
}

// DefaultConfig returns a Config with default settings for logging.
func DefaultConfig() Config {
	c := Config{
		lvl:                   new(slog.LevelVar),
		AddSource:             false,
		AttrProcessorPoolSize: DefaultAttrProcessorPoolSize,
		Adapter:               DefaultAdapterConfig(),
		Level:                 LevelInfo,
		NoTimestamp:           false,
		SetSlogOutput:         true,
		TimeFormat:            "15:04:05.000",
		TimeLocation:          time.Local,
	}
	return c
}

// ReplaceAttr adds a function to transform attributes, chaining with existing ones.
// Ignored if the Config is sealed.
func (c *Config) ReplaceAttr(f ReplaceAttrFunc) {
	if c.sealed {
		return
	}
	prev := c.replaceAttr
	if prev == nil {
		c.replaceAttr = f
		return
	}
	c.replaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		a = prev(groups, a)
		if a.Key == "" {
			return a
		}
		return f(groups, a)
	}
}

// HandlerOptions returns slog.HandlerOptions based on the Config.
func (c *Config) HandlerOptions() *slog.HandlerOptions {
	if !c.sealed {
		c.seal()
	}
	opts := &slog.HandlerOptions{
		AddSource: c.AddSource,
		Level:     c.lvl,
	}

	opts.ReplaceAttr = ReplaceAttrDefault(c.TimeLocation, c.TimeFormat)

	return opts
}

// seal finalizes the Config, applying level, adapter, and attribute settings.
func (c *Config) seal() {
	if c.lvl == nil {
		c.lvl = new(slog.LevelVar)
	}
	c.lvl.Set(c.Level.Level())
	c.Adapter = c.Adapter.seal()
	// ensure omit replacer
	if c.Omit != nil {
		if !c.checkReplaceAttr(c.Omit[0], slog.StringValue("omit-value"), "") {
			c.ReplaceAttr(ReplaceAttrOmit(c.Omit))
		}
	}

	// ensure secret replacer
	if c.Secrets != nil {
		if !c.checkReplaceAttr(c.Secrets[0], slog.StringValue("secret-value"), secretAttrValue) {
			c.ReplaceAttr(ReplaceAttrSecrets(c.Secrets))
		}
	}
	if c.NoTimestamp {
		c.TimeFormat = ""
	}
	c.sealed = true
}

// checkReplaceAttr verifies if the replaceAttr function produces the expected output.
func (c *Config) checkReplaceAttr(key string, val slog.Value, expected string) bool {
	attr := slog.Attr{Key: key, Value: val}
	if c.replaceAttr == nil {
		return false
	}
	a := c.replaceAttr(nil, attr)
	return a.String() == expected
}

// AdapterConfig defines settings for buffered adapters, controlling buffering and retry behavior.
type AdapterConfig struct {
	BufferSize    int           // Buffer size (adjusted to power of 2).
	Policy        AdapterPolicy // Policy for full buffers (Block or Drop).
	BatchSize     int           // Max records per batch.
	FlushInterval time.Duration // Interval for flushing batches (0 = immediate).
	FlushTimeout  time.Duration // Timout after flush fails if not completed
	MaxRetries    int           // Max retries for AdapterPolicyBlock.
	RetryTimeout  time.Duration // Max timeout for retries in AdapterPolicyBlock.
}

// seal finalizes the AdapterConfig, ensuring BufferSize is a power of 2.
func (bc *AdapterConfig) seal() AdapterConfig {
	if bc.BufferSize == 0 {
		bc.BufferSize = DefaultAdapterBufferSize
	} else {
		bc.BufferSize = int(bitutils.NextPowerOfTwo(uint64(bc.BufferSize)))
	}
	return *bc
}

// DefaultAdapterConfig returns an AdapterConfig with default settings.
func DefaultAdapterConfig() AdapterConfig {
	return AdapterConfig{
		BufferSize:    DefaultAdapterBufferSize,
		Policy:        AdapterPolicyBlock,
		BatchSize:     DefaultAdapterBatchSize,
		FlushInterval: DefaultAdapterFlushInterval,
		FlushTimeout:  DefaultAdapterFlushTimeout,
		MaxRetries:    DefaultAdapterMaxRetries,
		RetryTimeout:  DefaultAdapterRetryTimeout,
	}
}

// ReplaceAttrDefault transforms level and time attributes for slog handlers.
// It converts levels to custom strings and formats times based on Config.
func ReplaceAttrDefault(timeLocation *time.Location, timeFormat string) ReplaceAttrFunc {
	return func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.LevelKey:
			slvl := a.Value.Any().(slog.Level)
			a.Value = slog.StringValue(Level(slvl).String())
		case slog.TimeKey:
			if timeFormat == "" {
				return slog.Attr{}
			}
			t := a.Value.Time()
			a.Value = slog.StringValue(t.In(timeLocation).Format(timeFormat))
		}
		return a
	}
}

// ReplaceAttrOmit creates a ReplaceAttrFunc to omit specified attribute keys.
func ReplaceAttrOmit(omit []string) ReplaceAttrFunc {
	return func(groups []string, a slog.Attr) slog.Attr {
		if slices.Contains(omit, a.Key) {
			return slog.Attr{}
		}
		return a
	}
}

// ReplaceAttrSecrets creates a ReplaceAttrFunc to redact specified attribute keys.
func ReplaceAttrSecrets(secrets []string) ReplaceAttrFunc {
	return func(groups []string, a slog.Attr) slog.Attr {
		if slices.Contains(secrets, a.Key) {
			return slog.Attr{Key: a.Key, Value: secretSlogAttrValue}
		}
		return a
	}
}

// ReplaceAttrTime creates a ReplaceAttrFunc to format time attributes.
func ReplaceAttrTime(timeLocation *time.Location, timeFormat string) ReplaceAttrFunc {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			if timeFormat == "" {
				return slog.Attr{}
			}
			t := a.Value.Time()
			a.Value = slog.StringValue(t.In(timeLocation).Format(timeFormat))
		}
		return a
	}
}

// ReplaceAttrLevel creates a ReplaceAttrFunc to format level attributes.
func ReplaceAttrLevel(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		slvl := a.Value.Any().(slog.Level)
		a.Value = slog.StringValue(Level(slvl).String())
	}
	return a
}
