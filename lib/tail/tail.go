// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

/*
Package tail provides functionality for tailing files and readers, similar to the Unix 'tail -f' command.
It uses Go's iterator pattern to provide a composable and memory-efficient way to stream lines from files
and readers, with support for file rotation detection and context-based cancellation.

The package offers two main approaches:
  - iter.Seq2[string, error] for explicit error handling during iteration
  - iter.Seq[string] for simpler cases where iteration stops on the first error

# Basic Usage

To tail a file and handle errors explicitly:

	ctx := context.Background()
	for line, err := range tail.TailFile(ctx, "app.log", 10) {
		if err != nil {
			log.Printf("Error: %v", err)
			break
		}
		fmt.Println(line)
	}

To tail a file with automatic error handling (stops on first error):

	for line := range tail.TailLines(ctx, "app.log", 10) {
		fmt.Println(line)
	}

# File Rotation Support

TailFile automatically detects file rotation (rename/remove events) and reopens the new file:

	// This will continue working even if app.log is rotated
	for line := range tail.TailLines(ctx, "app.log", 0) {
		processLogLine(line)
	}

# Reader Support

For non-file readers, use TailReader:

	reader := strings.NewReader("line1\nline2\nline3\n")
	for line := range tail.TailReaderLines(ctx, reader, 2) {
		fmt.Println(line) // Prints: line2, line3
	}

# Initial Lines Parameter

The parameter 'n' controls how many recent lines to read initially:
  - n > 0: Read the last n lines initially, then stream new lines
  - n = 0: Read the last 10 lines initially (default), then stream new lines
  - n = -1: Skip initial lines, only stream new lines as they're written

# Composition with Standard Library

The iterators work seamlessly with Go's standard library functions:

	// Collect all lines into a slice
	lines := slices.Collect(tail.TailLines(ctx, "app.log", 5))

	// Filter lines while tailing
	errorLines := slices.Filter(tail.TailLines(ctx, "app.log", -1), func(line string) bool {
		return strings.Contains(line, "ERROR")
	})
	for line := range errorLines {
		alert(line)
	}

# Context Cancellation

All functions respect context cancellation for graceful shutdown:

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for line := range tail.TailLines(ctx, "app.log", 0) {
		if shouldStop(line) {
			cancel() // Will stop the iteration
			break
		}
		process(line)
	}

# Performance Considerations

The package is designed for efficiency:
  - Uses buffered I/O for reading
  - Implements debouncing for rapid file writes (10ms)
  - Efficiently seeks backward to read last N lines from large files
  - Memory usage scales with the number of initial lines requested, not file size
  - Pull-based iteration provides natural backpressure

# Error Handling

Errors can be handled in two ways:

 1. Explicit error handling with iter.Seq2[string, error]:
    for line, err := range tail.TailFile(ctx, "app.log", 10) {
    if err != nil {
    // Handle specific error
    continue // or break
    }
    // Process line
    }

 2. Automatic error handling with iter.Seq[string] (stops on first error):
    for line := range tail.TailLines(ctx, "app.log", 10) {
    // Process line (iteration stops if any error occurs)
    }

# Supported Platforms

The package works on all platforms supported by Go, with file rotation detection
using the fsnotify package. File birth time detection may have varying accuracy
depending on the filesystem and platform.
*/
package tail

import (
	"bufio"
	"context"
	"io"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/happy-sdk/happy/pkg/bytesize"
)

const (
	DefaultInitialLines = 10
	// 8 KiB buffer - better for modern structured logging
	// Handles most JSON logs without reallocation
	DefaultBufferSize    = 8 * bytesize.KiB
	DefaultMaxBufferSize = bytesize.MiB
)

var (
	currentBufferSize    atomic.Uint64
	currentMaxBufferSize atomic.Uint64
)

func init() {
	SetBufferSizes(DefaultBufferSize, DefaultMaxBufferSize)
}

func SetBufferSizes(bufferSize, maxBufferSize bytesize.IECSize) {
	currentBufferSize.Store(bufferSize.Bytes())
	currentMaxBufferSize.Store(maxBufferSize.Bytes())
}

func BufferSizes() (bufferSize, maxBufferSize bytesize.IECSize) {
	bufferSize = bytesize.IECSize(currentBufferSize.Load())
	maxBufferSize = bytesize.IECSize(currentMaxBufferSize.Load())
	return
}

// Pool for reusable byte slices to reduce allocations in scanners
var bufferPool = sync.Pool{
	New: func() any {
		bufferSize, _ := BufferSizes()
		buf := make([]byte, bufferSize.Bytes())
		return &buf
	},
}

// newScannerWithPooledBuffer creates a scanner with a pooled buffer for efficiency
func newScannerWithPooledBuffer(r io.Reader) (*bufio.Scanner, func()) {
	bufPtr := bufferPool.Get().(*[]byte)
	scanner := bufio.NewScanner(r)

	bufferSize, maxBufferSize := BufferSizes()

	// Allow growth up to maxBufferSize for very large log entries
	// e.g. (stack traces, etc.)
	scanner.Buffer(*bufPtr, int(maxBufferSize.Bytes()))

	// Return cleanup function
	cleanup := func() {
		// Reset buffer size if it grew beyond reasonable size (>128 KiB)
		if cap(*bufPtr) > int(128*bytesize.KiB) {
			*bufPtr = make([]byte, int(bufferSize))
		} else {
			// Keep existing capacity, just reset length
			*bufPtr = (*bufPtr)[:min(len(*bufPtr), int(bufferSize))]
		}
		bufferPool.Put(bufPtr)
	}

	return scanner, cleanup
}

// TailFile watches the specified file for changes and returns an iterator over lines.
// The parameter n specifies the number of most recent lines to read initially (default 10).
// If n = -1, no initial lines are read, and only new lines are streamed.
// The iterator stops when the context is canceled or an error occurs.
func TailFile(ctx context.Context, filePath string, n int) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		// Initialize file watcher
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			yield("", err)
			return
		}
		defer watcher.Close()

		// Resolve absolute path to handle renames
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			yield("", err)
			return
		}

		// Open the file
		file, err := os.Open(absPath)
		if err != nil {
			yield("", err)
			return
		}
		defer file.Close()

		// Read initial lines if n >= 0
		if n >= 0 {
			if n == 0 {
				n = DefaultInitialLines
			}
			lines, err := readLastLines(file, n)
			if err != nil {
				yield("", err)
				return
			}
			// Yield initial lines
			for _, line := range lines {
				if !yield(line, nil) {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		} else {
			// Seek to the end if n = -1
			_, err = file.Seek(0, io.SeekEnd)
			if err != nil {
				yield("", err)
				return
			}
		}

		// Add file to watcher
		err = watcher.Add(absPath)
		if err != nil {
			yield("", err)
			return
		}

		currentFile := file
		lastWrite := time.Now()
		var lastSize int64 // Track file size

		// Get initial file size
		if info, err := currentFile.Stat(); err == nil {
			lastSize = info.Size()
		}

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					yield("", fsnotify.ErrEventOverflow)
					return
				}
				if event.Has(fsnotify.Write) {
					// Debounce rapid writes
					if time.Since(lastWrite) < 10*time.Millisecond {
						continue
					}
					lastWrite = time.Now()

					// Check for truncation by comparing current file size
					info, err := currentFile.Stat()
					if err != nil {
						yield("", err)
						return
					}
					currentSize := info.Size()
					if currentSize < lastSize {
						// Truncation detected: seek to start
						_, err := currentFile.Seek(0, io.SeekStart)
						if err != nil {
							yield("", err)
							return
						}
					}
					lastSize = currentSize

					// Read new lines
					scanner, cleanup := newScannerWithPooledBuffer(currentFile)
					defer cleanup()
					for scanner.Scan() {
						if !yield(scanner.Text(), nil) {
							return
						}
						select {
						case <-ctx.Done():
							return
						default:
						}
					}
					if err := scanner.Err(); err != nil && err != io.EOF {
						yield("", err)
						return
					}
				}
				if event.Has(fsnotify.Rename | fsnotify.Remove) {
					// Handle log rotation: re-open the new file
					currentFile.Close()
					newFile, err := os.Open(absPath)
					if err != nil {
						yield("", err)
						return
					}
					currentFile = newFile

					// Start reading from beginning of new file (don't seek to end!)
					// The new file may already contain log entries we need to process
					info, err := currentFile.Stat()
					if err != nil {
						yield("", err)
						return
					}
					lastSize = info.Size()

					// Read any existing content in the new file
					if lastSize > 0 {
						scanner, cleanup := newScannerWithPooledBuffer(currentFile)
						for scanner.Scan() {
							if !yield(scanner.Text(), nil) {
								cleanup()
								return
							}
							select {
							case <-ctx.Done():
								cleanup()
								return
							default:
							}
						}
						if err := scanner.Err(); err != nil && err != io.EOF {
							cleanup()
							yield("", err)
							return
						}
						cleanup()
					}

					err = watcher.Add(absPath)
					if err != nil {
						yield("", err)
						return
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				yield("", err)
				return
			}
		}
	}
}

// TailReader tails lines from an io.Reader and returns an iterator over lines.
// If the reader implements io.ReadSeeker, it seeks to read the last n lines efficiently.
// Otherwise, it buffers lines in memory. The parameter n specifies the number of most
// recent lines to send initially (default 10). If n = -1, no initial lines are sent.
func TailReader(ctx context.Context, r io.Reader, n int) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if n == 0 {
			n = 10 // Default to 10 lines
		}

		// Check if the reader is an io.ReadSeeker
		if rs, ok := r.(io.ReadSeeker); ok && n >= 0 {
			lines, err := readLastLines(rs, n)
			if err != nil {
				yield("", err)
				return
			}

			// Yield initial lines
			for _, line := range lines {
				if !yield(line, nil) {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}

			// Stream new lines
			scanner, cleanup := newScannerWithPooledBuffer(rs)
			defer cleanup()
			for scanner.Scan() {
				if !yield(scanner.Text(), nil) {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				yield("", err)
			}
			return
		}

		// Handle non-seekable io.Reader
		scanner, cleanup := newScannerWithPooledBuffer(r)
		defer cleanup()
		var lines []string
		if n >= 0 {
			// Buffer last n lines in memory
			lines = slices.Grow(lines, n) // Pre-allocate for efficiency
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
				if len(lines) > n {
					lines = lines[1:]
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				yield("", err)
				return
			}
			// Yield initial lines
			for _, line := range lines {
				if !yield(line, nil) {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		} else {
			// Stream new lines only (n = -1)
			for scanner.Scan() {
				if !yield(scanner.Text(), nil) {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
			if err := scanner.Err(); err != nil && err != io.EOF {
				yield("", err)
			}
		}
	}
}

// TailLines returns an iterator that yields only the lines (no errors).
// Any error encountered will cause the iterator to stop.
func TailLines(ctx context.Context, filePath string, n int) iter.Seq[string] {
	return func(yield func(string) bool) {
		for line, err := range TailFile(ctx, filePath, n) {
			if err != nil {
				return // Stop on error
			}
			if !yield(line) {
				return
			}
		}
	}
}

// TailReaderLines returns an iterator that yields only the lines from a reader (no errors).
// Any error encountered will cause the iterator to stop.
func TailReaderLines(ctx context.Context, r io.Reader, n int) iter.Seq[string] {
	return func(yield func(string) bool) {
		for line, err := range TailReader(ctx, r, n) {
			if err != nil {
				return // Stop on error
			}
			if !yield(line) {
				return
			}
		}
	}
}

// ReadLastLines returns an iterator over the last n lines of a ReadSeeker.
// This is a pure iterator version without context cancellation.
func ReadLastLines(rs io.ReadSeeker, n int) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		lines, err := readLastLines(rs, n)
		if err != nil {
			yield("", err)
			return
		}
		for _, line := range lines {
			if !yield(line, nil) {
				return
			}
		}
	}
}

// readLastLines reads the last n lines from an io.ReadSeeker. It seeks to the end
// and scans backward to collect lines, restoring the seeker to the end afterward.
// It assumes the seeker is positioned arbitrarily.
func readLastLines(rs io.ReadSeeker, n int) ([]string, error) {
	// Seek to end to get file size
	endPos, err := rs.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if endPos == 0 {
		return nil, nil // Empty file
	}

	var lines []string
	buf := make([]byte, 4096) // 4KB buffer for reading backward
	pos := endPos
	remainder := ""

	for pos > 0 && len(lines) < n {
		// Determine how much to read (up to buffer size)
		readSize := int64(len(buf))
		readSize = min(readSize, pos)

		pos -= readSize

		// Seek to position
		_, err := rs.Seek(pos, io.SeekStart)
		if err != nil {
			return nil, err
		}

		// Read chunk
		nRead, err := rs.Read(buf[:readSize])
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Process chunk in reverse
		chunk := string(buf[:nRead])
		linesChunk := strings.Split(remainder+chunk, "\n")
		remainder = linesChunk[0]  // Partial line at start
		newLines := linesChunk[1:] // Complete lines

		// Prepend new lines (since reading backward)
		for i := len(newLines) - 1; i >= 0 && len(lines) < n; i-- {
			if newLines[i] != "" { // Skip empty lines from split
				lines = append([]string{newLines[i]}, lines...)
			}
		}
	}

	// Handle case where file has fewer lines than n
	if remainder != "" && len(lines) < n {
		lines = append([]string{remainder}, lines...)
	}

	// Seek back to end
	_, err = rs.Seek(endPos, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return lines[:min(len(lines), n)], nil
}
