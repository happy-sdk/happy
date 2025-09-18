// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"bytes"
	"io"
	"sync"
)

// Buffer is a concurrency-safe buffer designed for application developers to use
// as an io.Writer for logging adapters. It implements io.Writer, io.Reader,
// io.Closer, and sync.Locker, ensuring thread-safe operations. Unlike
// bytes.Buffer, it prevents data races when used with logging adapters that take
// ownership of the writer. Use this when passing a buffer to a logging adapter to
// safely accumulate and read log data concurrently.
//
// Example:
//
//	buf := logging.NewBuffer()
//	logger := someAdapter(buf)
//	buf.Write([]byte("log message"))
//	data, _ := buf.ReadAll()
type Buffer struct {
	buf bytes.Buffer
	sync.RWMutex
}

// NewBuffer creates a new concurrency-safe Buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// Write writes bytes to the buffer, thread-safe.
func (b *Buffer) Write(p []byte) (n int, err error) {
	b.Lock()
	defer b.Unlock()
	return b.buf.Write(p)
}

// Read reads bytes from the buffer, thread-safe.
func (b *Buffer) Read(p []byte) (n int, err error) {
	b.RLock()
	defer b.RUnlock()
	return b.buf.Read(p)
}

// ReadAll reads all data from the buffer, thread-safe, returning a copy of the
// buffer's contents.
func (b *Buffer) ReadAll() ([]byte, error) {
	b.RLock()
	defer b.RUnlock()
	return io.ReadAll(&b.buf)
}

// Reset clears the buffer's contents, thread-safe.
func (b *Buffer) Reset() {
	b.Lock()
	defer b.Unlock()
	b.buf.Reset()
}

func (b *Buffer) Len() int {
	b.RLock()
	defer b.RUnlock()
	return b.buf.Len()
}

func (b *Buffer) Cap() int {
	b.RLock()
	defer b.RUnlock()
	return b.buf.Cap()
}

func (b *Buffer) String() string {
	strb, _ := b.ReadAll()
	return string(strb)
}

// LineBuffer is a pooled byte buffer for adapter developers to efficiently build
// log lines from records without frequent allocations. It uses a sync.Pool to
// reuse buffers, reducing memory overhead. Use this instead of bytes.Buffer in
// adapter implementations for constructing log lines. Buffers should be returned
// to the pool with Free() after use to maintain efficiency.
//
// Example:
//
//	buf := logging.NewLineBuffer()
//	buf.WriteString("log line")
//	adapter.Write(buf)
//	buf.Free()
type LineBuffer []byte

// lineBufPool manages a pool of LineBuffers with an initial capacity of 1KB.
var lineBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return (*LineBuffer)(&b)
	},
}

// NewLineBuffer retrieves a LineBuffer from the pool with an initial capacity of
// 1KB. Always call Free() when done to return the buffer to the pool.
func NewLineBuffer() *LineBuffer {
	return lineBufPool.Get().(*LineBuffer)
}

// Free returns the LineBuffer to the pool if its capacity is ≤ 16KB, resetting
// its length to 0. Larger buffers are discarded to prevent excessive memory use.
func (b *LineBuffer) Free() {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		lineBufPool.Put(b)
	}
}

// Reset clears the buffer's contents by setting its length to 0.
func (b *LineBuffer) Reset() {
	b.SetLen(0)
}

// Write appends bytes to the buffer, implementing io.Writer.
func (b *LineBuffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

// WriteString appends a string to the buffer.
func (b *LineBuffer) WriteString(s string) (int, error) {
	*b = append(*b, s...)
	return len(s), nil
}

// WriteByte appends a single byte to the buffer.
func (b *LineBuffer) WriteByte(c byte) error {
	*b = append(*b, c)
	return nil
}

// String returns the buffer's contents as a string.
func (b *LineBuffer) String() string {
	return string(*b)
}

// Len returns the current length of the buffer.
func (b *LineBuffer) Len() int {
	return len(*b)
}

// SetLen sets the buffer's length, truncating or zero-extending as needed.
func (b *LineBuffer) SetLen(n int) {
	*b = (*b)[:n]
}
