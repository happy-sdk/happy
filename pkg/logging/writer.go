// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022-2025 The Happy SDK Authors

package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

// Writer wraps an io.Writer with atomic updates for safe concurrent use.
// It supports swapping writers and implements io.Closer to close the underlying writer.
// The Writer takes ownership of the provided writer, closing it when swapped or adapter is disposed,
// except for os.Stdout and os.Stderr, which are not closed.
type Writer struct {
	mu sync.Mutex
	w  atomic.Pointer[io.Writer]
}

// NewWriter creates a Writer wrapping the given io.Writer, taking ownership.
// The wrapped writer will be closed when the Writer is closed or swapped,
// unless it is os.Stdout or os.Stderr.
func NewWriter(w io.Writer) *Writer {
	wr := &Writer{}
	wr.w.Store(&w)
	return wr
}

// Write writes p to the current writer, ensuring thread-safe access.
// If the writer does not implement sync.Locker, a mutex guards the write.
// Returns ErrWriterIO wrapping any I/O error from the underlying writer.
func (w *Writer) Write(p []byte) (n int, err error) {
	writer := *w.w.Load()
	if _, ok := writer.(sync.Locker); !ok {
		w.mu.Lock()
		defer w.mu.Unlock()
	}
	n, err = writer.Write(p)
	if err != nil {
		return n, fmt.Errorf("%w: %w", ErrWriterIO, err)
	}
	return n, nil
}

// Close closes the current writer if it implements io.Closer.
// os.Stdout and os.Stderr are not closed. Safe for concurrent use.
func (w *Writer) Close() error {
	writer := *w.w.Load()
	if writer == os.Stdout || writer == os.Stderr {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if closer, ok := writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Swap replaces the current writer with a new one, closing the old writer.
// os.Stdout and os.Stderr are not closed. Returns any error from closing.
func (w *Writer) Swap(new io.Writer) error {
	err := w.Close()
	w.w.Store(&new)
	return err
}

// Get returns the current writer. Safe for concurrent access.
func (w *Writer) Get() io.Writer {
	return *w.w.Load()
}
