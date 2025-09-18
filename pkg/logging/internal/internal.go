// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

// Package internal contains experimental and undecided APIs for the logging package.
// These APIs support internal features (e.g., QueueLogger, HTTP logging, adapter type
// identification) but are not part of the public github.com/happy-sdk/happy/pkg/logging
// API. They are subject to change without notice and intended for package maintainers only.
package internal

import (
	"log/slog"
	"math"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/happy-sdk/happy/pkg/bitutils"
)

const (
	// HttpRecordKey is the key for HTTP record attributes.
	HttpRecordKey   = "__logging_http"
	httpRecordLevel = slog.Level(math.MaxInt - 2) // same as logging.LevelOut Level = Level(math.MaxInt - 2)
)

// BufferedAdapter extends Adapter with internal buffered features.
type BufferedAdapter[R any] interface {
	// GetBufferedAdapterName returns the name of the buffered adapter type.
	GetBufferedAdapterName() string
	// AcceptsHTTP reports if the adapter accepts HTTP records.
	AcceptsHTTP() bool

	RecordHandle(record R) error
}

// HttpRecord holds HTTP-specific log data.
type HttpRecord struct {
	Method string
	Code   int
	Path   string
}

// NewHttpRecord creates a slog.Record for an HTTP event.
func NewHttpRecord(t time.Time, method string, statusCode int, path string, args ...any) slog.Record {
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(t, httpRecordLevel, "", pcs[0])

	r.AddAttrs(slog.Any(HttpRecordKey, HttpRecord{
		Method: method,
		Code:   statusCode,
		Path:   path,
	}))
	r.Add(args...)
	return r
}

// RingBuffer is a high-performance lock-free ring buffer for generic records.
type RingBuffer[R any] struct {
	// Records slice 24 bytes
	buffer []R
	// Buffer mask (BufferSize - 1) atomic uint ensures 8 byte alignment
	// and hot path optimization, being always in same cache line
	mask atomic.Uint64
	_    [32]byte      // 1 cache line
	head atomic.Uint64 // Write position
	_    [56]byte      // Cache line padding
	tail atomic.Uint64 // Read position
	_    [56]byte      // Cache line padding
}

// NewRingBuffer creates a RingBuffer of size (power of 2)
func NewRingBuffer[R any](size uint64) *RingBuffer[R] {
	size = bitutils.NextPowerOfTwo(size)
	buf := &RingBuffer[R]{
		buffer: make([]R, size),
	}
	buf.mask.Store(size - 1)
	return buf
}

// Len returns the current number of items in the buffer.
func (b *RingBuffer[R]) Len() int {
	head := b.head.Load()
	tail := b.tail.Load()
	return int(head - tail)
}

// Cap returns the buffer capacity.
func (b *RingBuffer[R]) Cap() int {
	return len(b.buffer)
}

// Add appends an item to the buffer (optimized with relaxed ordering).
func (b *RingBuffer[R]) Add(r R) {
	head := b.head.Load()
	b.buffer[head&b.mask.Load()] = r
	// Use relaxed memory ordering for better performance
	// Only use if you can guarantee ordering through other means
	b.head.Store(head + 1)
}

// AddUnsafe is a faster version that doesn't check for overflow
// Use only when you can guarantee the buffer won't overflow
func (b *RingBuffer[R]) AddUnsafe(r R) {
	head := b.head.Load()
	*(*R)(unsafe.Pointer(uintptr(unsafe.Pointer(&b.buffer[0])) +
		uintptr(head&b.mask.Load())*unsafe.Sizeof(r))) = r
	b.head.Store(head + 1)
}

// Take retrieves and removes the oldest item.
func (b *RingBuffer[R]) Take() (record R, loaded bool) {
	head := b.head.Load()
	tail := b.tail.Load()
	if head == tail {
		return
	}
	record = b.buffer[tail&b.mask.Load()]
	loaded = true
	b.tail.Store(tail + 1)
	return
}

// TakeUnsafe is a faster version using unsafe pointer
func (b *RingBuffer[R]) TakeUnsafe() (record R, loaded bool) {
	head := b.head.Load()
	tail := b.tail.Load()
	if head == tail {
		return
	}
	record = *(*R)(unsafe.Pointer(uintptr(unsafe.Pointer(&b.buffer[0])) +
		uintptr(tail&b.mask.Load())*unsafe.Sizeof(record)))
	loaded = true
	b.tail.Store(tail + 1)
	return
}

// TakeBatch retrieves up to count items (optimized version).
func (b *RingBuffer[R]) TakeBatch(count int) []R {
	head := b.head.Load()
	tail := b.tail.Load()
	available := head - tail

	if available == 0 {
		return nil
	}

	// Determine actual batch size
	actualBatchSize := min(int(available), count)
	batch := make([]R, actualBatchSize)

	mask := b.mask.Load()
	startIdx := tail & mask
	endIdx := (tail + uint64(actualBatchSize) - 1) & mask

	if startIdx <= endIdx {
		copy(batch, b.buffer[startIdx:startIdx+uint64(actualBatchSize)])
	} else {
		firstPart := uint64(len(b.buffer)) - startIdx
		copy(batch[:firstPart], b.buffer[startIdx:])
		copy(batch[firstPart:], b.buffer[:uint64(actualBatchSize)-firstPart])
	}

	b.tail.Store(tail + uint64(actualBatchSize))
	return batch
}

// Drain retrieves and removes all available items.
func (b *RingBuffer[R]) Drain() []R {
	head := b.head.Load()
	tail := b.tail.Load()
	available := head - tail

	if available == 0 {
		return nil
	}

	batch := make([]R, available)

	mask := b.mask.Load()
	startIdx := tail & mask
	endIdx := (head - 1) & mask

	if startIdx <= endIdx || available <= mask+1-startIdx {
		// Contiguous copy
		if available <= mask+1-startIdx {
			copy(batch, b.buffer[startIdx:startIdx+available])
		} else {
			copy(batch, b.buffer[startIdx:startIdx+available])
		}
	} else {
		// Wrapped copy
		firstPart := uint64(len(b.buffer)) - startIdx
		copy(batch[:firstPart], b.buffer[startIdx:])
		copy(batch[firstPart:], b.buffer[:available-firstPart])
	}

	b.tail.Store(head)
	return batch
}

// Empty reports if the buffer is empty.
func (b *RingBuffer[R]) Empty() bool {
	return b.head.Load() == b.tail.Load()
}

// IsFull reports if the buffer is full (useful for bounded operations)
func (b *RingBuffer[R]) IsFull() bool {
	return b.Len() >= len(b.buffer)
}

// TakePreallocatedBatch reuses a provided slice to avoid allocations
func (b *RingBuffer[R]) TakePreallocatedBatch(batch []R) []R {
	head := b.head.Load()
	tail := b.tail.Load()
	available := head - tail

	if available == 0 {
		return batch[:0]
	}

	count := min(int(available), len(batch))
	batch = batch[:count]
	mask := b.mask.Load()
	startIdx := tail & mask
	endIdx := (tail + uint64(count) - 1) & mask

	if startIdx <= endIdx {
		copy(batch, b.buffer[startIdx:startIdx+uint64(count)])
	} else {
		firstPart := uint64(len(b.buffer)) - startIdx
		copy(batch[:firstPart], b.buffer[startIdx:])
		copy(batch[firstPart:], b.buffer[:uint64(count)-firstPart])
	}

	b.tail.Store(tail + uint64(count))
	return batch
}
