// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/happy-sdk/happy/pkg/bitutils"
	"github.com/happy-sdk/happy/pkg/logging/internal"
)

// BufferedAdapter provides high-performance buffering for adapters used by Logger.
// It wraps an Adapter implementing slog.Handler, using a lock-free ring buffer to
// handle burst logging without blocking the caller.
type BufferedAdapter[A Adapter] struct {
	config             AdapterConfig                // Configuration
	adapter            Adapter                      // Underlying adapter
	adapterName        string                       // Name of the adapter
	adapterAcceptsHTTP bool                         // Whether adapter supports HTTP
	droppedCounter     *expvar.Int                  // Dropped records counter
	droppedCount       atomic.Int64                 // Dropped records counter
	disposed           atomic.Bool                  // Disposal flag
	queueLen           atomic.Int64                 // Length of unporcessed records
	err                atomic.Value                 // Last error
	records            *internal.RingBuffer[Record] // Lock-free ring buffer
	wg                 sync.WaitGroup               // Wait for worker goroutine

	workerStart      sync.Once
	stopCh           chan struct{} // Shutdown signal
	flushCh          chan int64    // Flush signal
	flushCompletedCh chan int64    // Flush cycle completed signal
}

// NewBufferedAdapter creates a BufferedAdapter wrapping the given adapter.
func NewBufferedAdapter[A Adapter](adapter A, adapterConfig AdapterConfig, droppedCounter *expvar.Int) *BufferedAdapter[A] {
	// Ensure buffer size is power of 2
	if adapterConfig.BufferSize == 0 {
		adapterConfig.BufferSize = DefaultAdapterBufferSize
	} else if (adapterConfig.BufferSize & (adapterConfig.BufferSize - 1)) != 0 {
		adapterConfig.BufferSize = int(bitutils.NextPowerOfTwo(uint64(adapterConfig.BufferSize)))
	}
	if adapterConfig.BatchSize <= 0 {
		adapterConfig.BatchSize = DefaultAdapterBatchSize
	}
	if adapterConfig.MaxRetries <= 0 {
		adapterConfig.MaxRetries = 1
	}
	if adapterConfig.RetryTimeout <= 0 {
		adapterConfig.RetryTimeout = DefaultAdapterRetryTimeout
	}

	badapter := &BufferedAdapter[A]{
		config:           adapterConfig,
		adapter:          adapter,
		adapterName:      fmt.Sprintf("%T", adapter),
		records:          internal.NewRingBuffer[Record](uint64(adapterConfig.BufferSize)),
		droppedCounter:   droppedCounter,
		stopCh:           make(chan struct{}),
		flushCh:          make(chan int64),
		flushCompletedCh: make(chan int64),
	}
	_, badapter.adapterAcceptsHTTP = badapter.adapter.(AdapterWithHTTPHandle)
	if !badapter.adapterAcceptsHTTP {
		_, badapter.adapterAcceptsHTTP = badapter.adapter.(AdapterWithHTTPBatchHandle)
	}

	return badapter
}

// NewBufferedTextAdapter creates an AdapterComposer for a buffered text adapter.
func NewBufferedTextAdapter(w io.Writer, dropped *expvar.Int) *AdapterComposer[*BufferedAdapter[Adapter]] {
	return NewAdapter(w, func(ww *Writer, config Config) *BufferedAdapter[Adapter] {
		return NewBufferedAdapter(NewAdapterWithHandler(ww, slog.NewTextHandler).Compose(config), config.Adapter, dropped)
	})
}

// NewBufferedJSONAdapter creates an AdapterComposer for a buffered JSON adapter.
func NewBufferedJSONAdapter(w io.Writer, dropped *expvar.Int) *AdapterComposer[*BufferedAdapter[Adapter]] {
	return NewAdapter(w, func(ww *Writer, config Config) *BufferedAdapter[Adapter] {
		return NewBufferedAdapter(NewAdapterWithHandler(ww, slog.NewJSONHandler).Compose(config), config.Adapter, dropped)
	})
}

// GetBufferedAdapterName returns the name of the adapter being buffered.
func (b *BufferedAdapter[A]) GetBufferedAdapterName() string {
	return b.adapterName
}

// AcceptsHTTP returns true if the adapter accepts HTTP records.
func (b *BufferedAdapter[A]) AcceptsHTTP() bool {
	return b.adapterAcceptsHTTP
}

// Enabled checks if the underlying adapter handles the given level.
func (b *BufferedAdapter[A]) Enabled(ctx context.Context, l slog.Level) bool {
	if b.disposed.Load() {
		return false
	}
	return b.adapter.Enabled(ctx, l)
}

// WithAttrs returns a lightweight derived adapter with added attributes.
func (b *BufferedAdapter[A]) WithAttrs(attrs []slog.Attr) slog.Handler {
	if b.disposed.Load() {
		return b
	}
	return b.adapter.WithAttrs(attrs)
}

// WithGroup returns a lightweight derived adapter with the specified group name.
func (b *BufferedAdapter[A]) WithGroup(name string) slog.Handler {
	if b.disposed.Load() {
		return b
	}
	return b.adapter.WithGroup(name)
}

// Handle buffers a log record based on the configured policy.
func (b *BufferedAdapter[A]) Handle(ctx context.Context, record slog.Record) error {
	return b.RecordHandle(Record{Ctx: ctx, Record: record})
}

// Err returns the last error encountered by the adapter, or nil if none.
func (b *BufferedAdapter[A]) Err() error {
	if err, ok := b.err.Load().(*AdapterError); ok {
		return err.Err()
	}
	return nil
}

// Dropped returns the number of records dropped due to buffer overflow.
func (b *BufferedAdapter[A]) Dropped() int64 {
	return b.droppedCount.Load()
}

// RecordHandle handles a single record with retry semantics.
func (b *BufferedAdapter[A]) RecordHandle(record Record) error {
	if b.disposed.Load() {
		return ErrAdapterDisposed
	}
	if err := b.Err(); err != nil {
		return err
	}

	b.queueLen.Add(1)

	// Start worker
	b.workerStart.Do(func() {
		b.wg.Add(1)
		go b.worker()
		time.Sleep(time.Microsecond) // Ensure goroutine starts
	})

	if record.isHTTP && !b.adapterAcceptsHTTP {
		return nil
	}

	if err := b.tryToBuffer(record); err == nil {
		return nil
	} else if !errors.Is(err, ErrAdapterBufferFullRetry) {
		b.err.Store(NewAdapterError(err))
		return err
	}

	retryCtx, cancel := context.WithTimeout(context.Background(), b.config.RetryTimeout)
	defer cancel()
	startTime := time.Now()

	for attempt := range b.config.MaxRetries {
		if err := b.tryToBuffer(record); err == nil {
			return nil
		}

		backoffMs := 10 * attempt * attempt
		backoffDuration := min(time.Duration(backoffMs)*time.Millisecond, time.Second)
		elapsed := time.Since(startTime)

		if elapsed+backoffDuration >= b.config.RetryTimeout {
			err := fmt.Errorf("%w: timeout after %d/%d attempts in %s", ErrAdapterBufferFull, attempt+1, b.config.MaxRetries, elapsed)
			b.err.Store(NewAdapterError(err))
			b.drop()
			return err
		}

		select {
		case <-time.After(backoffDuration):
			continue
		case <-retryCtx.Done():
			err := fmt.Errorf("%w: retry failed after %d attempts elapsed %s", ErrAdapterBufferFull, attempt+1, time.Since(startTime))
			b.err.Store(NewAdapterError(err))
			b.drop()
		case <-b.stopCh:
			b.drop()
			return ErrAdapterDisposed
		}
	}
	err := fmt.Errorf("%w max attempts reached %d record dropped", ErrAdapterBufferFull, b.config.MaxRetries)
	b.drop()
	b.err.Store(NewAdapterError(err))
	return err
}

// Flush triggers processing of all buffered records and flushes the underlying adapter.
func (b *BufferedAdapter[A]) Flush() error {
	if b.disposed.Load() {
		return ErrAdapterDisposed
	}

	queueLen := b.queueLen.Load()
	//
	if queueLen == 0 {
		return nil
	}
	flushStarted := time.Now()

	fts := time.Now().UnixNano()
	b.flushCh <- fts
	timeout := time.After(b.config.FlushTimeout)
	for {
		select {
		case cfts := <-b.flushCompletedCh:
			if cfts == fts {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("flush timed out after %v", time.Since(flushStarted))
		}
	}
}

// Dispose stops the processing goroutine and disposes the underlying adapter.
func (b *BufferedAdapter[A]) Dispose() error {
	if b.disposed.Swap(true) {
		return b.Err()
	}

	close(b.stopCh)
	b.wg.Wait()
	close(b.flushCh)
	close(b.flushCompletedCh)

	if disposable, ok := b.adapter.(DisposableAdapter); ok {
		if err := disposable.Dispose(); err != nil {
			// debugLn("Dispose: adapter dispose error:", err)
			b.err.Store(NewAdapterError(err))
		}
	}
	return b.Err()
}

func (b *BufferedAdapter[A]) worker() {
	defer b.wg.Done()
	var ticker *time.Ticker
	var tickerCh <-chan time.Time
	if b.config.FlushInterval > 0 {
		ticker = time.NewTicker(b.config.FlushInterval)
		defer ticker.Stop()
		tickerCh = ticker.C
	} else {
		tickerCh = make(chan time.Time)
	}

	for {
		select {
		case <-tickerCh:
			available := b.records.Len()
			if available == 0 {
				continue
			}
			b.processRecordsBatch(b.records.TakeBatch(min(available, b.config.BatchSize)))
		case flushId, ok := <-b.flushCh:
			if !ok {
				continue
			}
			if b.records.Len() == 0 && b.queueLen.Load() == 0 {
				if flushId > 0 {
					b.flushCompletedCh <- flushId
				}
				continue
			}
			batch := make([]Record, b.config.BatchSize)

		collect:
			for b.records.Len() > 0 {
				batch = b.records.TakePreallocatedBatch(batch)
				b.processRecordsBatch(batch)
			}

			if b.queueLen.Load() > 0 {
				// there is more records to process
				runtime.Gosched()
				goto collect
			}
			if flushId > 0 {
				b.flushCompletedCh <- flushId
			}
		case <-b.stopCh:
			return
		}
	}
}

func (b *BufferedAdapter[A]) tryToBuffer(record Record) (err error) {
	if b.records.Len() < b.config.BufferSize {
		b.records.AddUnsafe(record)
		return
	}

	if b.config.Policy == AdapterPolicyDrop {
		b.drop()
		err := ErrAdapterBufferFull
		b.err.Store(NewAdapterError(err))
		return err
	}
	select {
	case b.flushCh <- 0:
	default: // flush signal already pending
	}
	return ErrAdapterBufferFullRetry
}

func (b *BufferedAdapter[A]) drop() {
	b.queueLen.Add(-1)
	if b.droppedCounter != nil {
		b.droppedCounter.Add(1)
	}
}

func (b *BufferedAdapter[A]) processRecordsBatch(records []Record) {
	defer b.queueLen.Add(-int64(len(records)))

	var (
		stdrec  []Record
		httprec []HttpRecord
	)

	// we know that records can not contain http records
	if b.adapterAcceptsHTTP {
		for _, r := range records {
			if r.isHTTP {
				httprec = append(httprec, HttpRecord{
					Ctx:    r.Ctx,
					Record: r.Record,
					Method: r.http.Method,
					Code:   r.http.Code,
					Path:   r.http.Path,
				})
			} else {
				stdrec = append(stdrec, r)
			}
		}
	} else {
		stdrec = records
		goto std
	}

	if len(httprec) > 0 {
		for _, hr := range httprec {
			if httpa, ok := b.adapter.(AdapterWithHTTPHandle); ok {
				if err := httpa.HTTP(hr.Ctx, hr.Method, hr.Code, hr.Path, hr.Record); err != nil {
					b.err.Store(NewAdapterError(err))
				}
			} else if httpAdapter, ok := b.adapter.(AdapterWithHTTPBatchHandle); ok {
				if err := httpAdapter.HTTPBatchHandle([]HttpRecord{
					{Ctx: hr.Ctx, Record: hr.Record, Method: hr.Method, Code: hr.Code, Path: hr.Path},
				}); err != nil {
					b.err.Store(NewAdapterError(err))
				}
			}
		}
	}

std:
	if badapter, ok := b.adapter.(AdapterWithBatchHandle); ok {
		if err := badapter.BatchHandle(stdrec); err != nil {
			b.err.Store(NewAdapterError(err))
			return
		}
	} else {
		// handlerer does not have batch handlerer use std handlerer
		for _, r := range stdrec {
			if err := b.adapter.Handle(r.Ctx, r.Record); err != nil {
				b.err.Store(NewAdapterError(err))
			}
		}
	}
}
