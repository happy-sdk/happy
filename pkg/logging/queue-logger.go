// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/happy-sdk/happy/pkg/bitutils"
)

// QueueLogger is a lightweight, slog-compatible logger that queues log records in a
// ring buffer with a Block policy, ensuring no logs are lost. It collects records without
// writing them until drained via Consume into a configured Logger. It can be set as
// slog.Default and reused after consumption.
//
// Usage:
//
//	queue := NewQueueLogger(1024) // ~0.7 MiB for 1024 records
//	queue.Info("early log", "key", "value")
//	config := DefaultConfig()
//	logger := New(config, NewBufferedTextAdapter(os.Stdout, nil))
//	logger.Consume(queue) // Drain queued records to logger
//	defer logger.Dispose()
type QueueLogger struct {
	*slog.Logger
	adapter *BufferedAdapter[*queueAdapter]
	queue   *queueAdapter
}

// NewQueueLogger creates a QueueLogger with the given buffer size (~0.7 MiB for 1024).
// The Block policy ensures no records are lost. Size must be a power of two.
func NewQueueLogger(size int) *QueueLogger {
	config := AdapterConfig{
		BufferSize:    int(bitutils.NextPowerOfTwo(uint64(size))),
		Policy:        AdapterPolicyBlock,
		BatchSize:     512,
		FlushInterval: 0,
		FlushTimeout:  DefaultAdapterFlushTimeout,
		MaxRetries:    DefaultAdapterMaxRetries,
		RetryTimeout:  DefaultAdapterRetryTimeout,
	}
	queue := &queueAdapter{
		buf: []Record{},
	}
	adapter := NewBufferedAdapter(queue, config, nil)
	ql := &QueueLogger{
		Logger:  slog.New(adapter),
		adapter: adapter,
		queue:   queue,
	}
	return ql
}

// Consume drains all queued records into the target Logger.
// Call *QueueLogger Dispose dont want to use it anymore to release
// intrnal BufferAdapter
// It returns the number of records consumed and any error encountered.
func (ql *QueueLogger) Consume(logger *Logger) (int, error) {
	if ql.queue.disposed.Load() {
		return 0, fmt.Errorf("%w: QueueLogger disposed", ErrAdapter)
	}
	ql.queue.mu.Lock()
	adapter := ql.adapter
	ql.queue.mu.Unlock()
	adapter.Flush()

	ql.queue.mu.Lock()
	defer ql.queue.mu.Unlock()

	records := ql.queue.buf

	processed, err := logger.handler.queueHandle(records)
	if err != nil {
		return processed, err
	}
	if err := logger.Flush(); err != nil {
		return processed, err
	}
	return processed, nil
}

func (ql *QueueLogger) Dispose() error {
	if ql.queue.disposed.Swap(true) {
		return nil
	}
	ql.queue.mu.Lock()
	defer ql.queue.mu.Unlock()

	_ = ql.adapter.Dispose()
	ql.adapter = nil
	ql.Logger = slog.New(DiscardAdapter)
	return nil
}

type queueAdapter struct {
	mu       sync.RWMutex
	err      atomic.Value
	buf      []Record
	disposed atomic.Bool
}

// Enabled return always true unless Disposed.
func (qa *queueAdapter) Enabled(ctx context.Context, level slog.Level) bool {
	return !qa.disposed.Load()
}

func (qa *queueAdapter) Handle(ctx context.Context, record slog.Record) error {
	if qa.disposed.Load() {
		return fmt.Errorf("%w: QueueLogger disposed", ErrAdapter)
	}
	qa.mu.Lock()
	defer qa.mu.Unlock()
	rec := Record{Ctx: ctx, Record: record}
	qa.buf = append(qa.buf, rec)
	return nil
}

func (qa *queueAdapter) HandlerBatch(records []Record) error {
	qa.mu.Lock()
	defer qa.mu.Unlock()
	qa.buf = append(qa.buf, records...)
	return nil
}

func (qa *queueAdapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return qa
}

func (qa *queueAdapter) WithGroup(name string) slog.Handler {
	return qa
}
