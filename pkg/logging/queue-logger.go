// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

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
func (ql *QueueLogger) Consume(queue *QueueLogger) (int, error) {
	if ql.queue.disposed.Load() {
		return 0, fmt.Errorf("%w: QueueLogger disposed", ErrAdapter)
	}

	if err := ql.adapter.Flush(); err != nil {
		return 0, err
	}
	if err := queue.adapter.Flush(); err != nil {
		return 0, err
	}

	ql.queue.mu.Lock()
	defer ql.queue.mu.Unlock()

	queue.queue.mu.Lock()
	records := queue.queue.buf
	queue.queue.mu.Unlock()

	ql.queue.buf = append(ql.queue.buf, records...)

	return len(records), nil
}

// LogDepth logs a message with additional context at a given depth.
// The depth is the number of stack frames to ascend when logging the message.
// It is useful only when AddSource is enabled.
func (ql *QueueLogger) LogDepth(depth int, lvl Level, msg string, attrs ...slog.Attr) error {
	if ql.queue.disposed.Load() {
		return fmt.Errorf("%w: QueueLogger disposed", ErrAdapter)
	}

	var pcs [1]uintptr
	runtime.Callers(depth+2, pcs[:])
	r := slog.NewRecord(time.Now(), slog.Level(lvl), msg, pcs[0])
	r.AddAttrs(attrs...)
	return ql.Handler().Handle(context.Background(), r)
}

func (ql *QueueLogger) Records() []Record {
	_ = ql.adapter.Flush()
	ql.queue.mu.Lock()
	defer ql.queue.mu.Unlock()

	records := ql.queue.buf
	return records
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
