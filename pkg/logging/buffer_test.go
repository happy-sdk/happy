// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package logging

import (
	"bytes"
	"sync"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestBufferConcurrency(t *testing.T) {
	buf := NewBuffer()
	defer buf.Reset()

	var wg sync.WaitGroup
	writeFn := func(data []byte) {
		defer wg.Done()
		_, err := buf.Write(data)
		testutils.NoError(t, err, "write should not error")
	}

	for range 10 {
		wg.Add(1)
		go writeFn([]byte("test"))
	}
	wg.Wait()

	data, err := buf.ReadAll()
	testutils.NoError(t, err, "ReadAll should not error")
	testutils.Assert(t, len(data) > 0, "buffer should contain data after writes")
}

func TestBufferReset(t *testing.T) {
	buf := NewBuffer()
	_, err := buf.Write([]byte("test data"))
	testutils.NoError(t, err, "write should not error")

	buf.Reset()
	data, err := buf.ReadAll()
	testutils.NoError(t, err, "ReadAll should not error")
	testutils.Assert(t, len(data) == 0, "buffer should be empty after reset")
}

func TestLineBuffer(t *testing.T) {
	b := NewLineBuffer()
	defer b.Free()
	n, err := b.WriteString("hello")
	testutils.NoError(t, err)
	testutils.Assert(t, n > 0)
	testutils.NoError(t, b.WriteByte(','))
	n2, err2 := b.Write([]byte(" world"))
	testutils.NoError(t, err2)
	testutils.Assert(t, n2 > 0)
	testutils.Equal(t, b.String(), "hello, world")

	b.Reset()
	testutils.Equal(t, 0, b.Len())
}

func TestLineBufferAlloc(t *testing.T) {
	got := int(testing.AllocsPerRun(5, func() {
		b := NewLineBuffer()
		defer b.Free()
		_, err := b.WriteString("not 1K worth of bytes")
		testutils.NoError(t, err)
	}))
	testutils.Assert(t, got < 2)
}

func TestLineBufferWriteMethods(t *testing.T) {
	b := NewLineBuffer()
	defer b.Free()

	// Test WriteString
	_, err := b.WriteString("hello")
	testutils.Assert(t, err == nil, "WriteString should not error")
	testutils.Assert(t, b.String() == "hello", "WriteString content mismatch")

	// Test WriteByte
	err = b.WriteByte(',')
	testutils.Assert(t, err == nil, "WriteByte should not error")
	testutils.Assert(t, b.String() == "hello,", "WriteByte content mismatch")

	// Test Write
	_, err = b.Write([]byte(" world"))
	testutils.Assert(t, err == nil, "Write should not error")
	testutils.Assert(t, b.String() == "hello, world", "Write content mismatch")
}

func TestLineBufferSetLen(t *testing.T) {
	b := NewLineBuffer()
	defer b.Free()

	n, err := b.WriteString("hello, world")
	testutils.NoError(t, err)
	testutils.Assert(t, n > 0)
	b.SetLen(5)
	testutils.Assert(t, b.String() == "hello", "SetLen should truncate to 'hello'")
	testutils.Assert(t, b.Len() == 5, "Len should be 5 after SetLen")

	b.SetLen(0)
	testutils.Assert(t, b.String() == "", "SetLen(0) should clear buffer")
	testutils.Assert(t, b.Len() == 0, "Len should be 0 after SetLen(0)")
}

func TestLineBufferPoolReuse(t *testing.T) {
	// Test pool reuse efficiency
	b1 := NewLineBuffer()
	n, err := b1.WriteString("test")
	testutils.NoError(t, err)
	testutils.Assert(t, n > 0)
	initialCap := cap(*b1)
	b1.Free()

	// Get a new buffer, should reuse the previous one
	b2 := NewLineBuffer()
	testutils.Assert(t, cap(*b2) == initialCap, "pooled buffer should have same capacity")
	testutils.Assert(t, b2.String() == "", "pooled buffer should be empty")
	b2.Free()
}

func TestLineBufferLargeBuffer(t *testing.T) {
	b := NewLineBuffer()
	defer b.Free()

	// Create a large buffer exceeding maxBufferSize (16KB)
	largeData := bytes.Repeat([]byte("x"), 17<<10)
	_, err := b.Write(largeData)
	testutils.Assert(t, err == nil, "Write large data should not error")
	testutils.Assert(t, b.Len() == 17<<10, "buffer length should match large data")

	// Free should not return large buffer to pool
	b.Free()

	// New buffer should not inherit large capacity
	b2 := NewLineBuffer()
	testutils.Assert(t, cap(*b2) <= 16<<10, "new buffer should not exceed max capacity")
	b2.Free()
}
