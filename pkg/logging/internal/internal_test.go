// SPDX-License-Identifier: Apache-2.0
// Copyright © 2018-2025 The Happy SDK Authors

package internal

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRingBufferAddTake(t *testing.T) {
	b := NewRingBuffer[int](8)
	b.Add(1)
	b.Add(2)
	if got := b.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
	v, ok := b.Take()
	if !ok || v != 1 {
		t.Fatalf("Take() = (%d, %v), want (1, true)", v, ok)
	}
	v, ok = b.Take()
	if !ok || v != 2 {
		t.Fatalf("Take() = (%d, %v), want (2, true)", v, ok)
	}
	if _, ok := b.Take(); ok {
		t.Fatal("Take() on empty buffer should report loaded=false")
	}
}

// TestRingBufferConcurrentProducersNoLostUpdates is a regression test for a
// data race in Add/AddUnsafe: head advancement used a non-atomic
// Load-then-Store sequence, so two concurrent producers could load the same
// head value, write into the same slot (silently dropping one record), and
// jointly advance head by only 1 instead of 2. That lost-update under-counts
// head relative to the number of successful Add calls, which downstream (in
// BufferedAdapter) desyncs a separately-tracked pending-count from what the
// ring buffer can ever actually yield, causing an indefinite busy-loop.
//
// This test runs many goroutines each adding a known number of items
// concurrently, drains them with a single consumer goroutine (mirroring the
// real multi-producer/single-consumer usage in BufferedAdapter), and asserts
// every single item is observed exactly once -- no drops, no duplicates.
func TestRingBufferConcurrentProducersNoLostUpdates(t *testing.T) {
	const (
		producers       = 50
		itemsPerProduce = 500
		total           = producers * itemsPerProduce
	)

	// Sized comfortably larger than total so producers never have to wait on
	// the (slow, busy-polling) consumer below; this test isolates the
	// lost-update race on Add/AddUnsafe, not buffer-full/backpressure
	// behavior (which is handled by callers, e.g. BufferedAdapter).
	b := NewRingBuffer[int64](1 << 17)

	var nextID atomic.Int64
	var wg sync.WaitGroup
	wg.Add(producers)
	for range producers {
		go func() {
			defer wg.Done()
			for range itemsPerProduce {
				b.AddUnsafe(nextID.Add(1))
			}
		}()
	}

	seen := make([]bool, total+1)
	var seenCount atomic.Int64
	done := make(chan struct{})
	go func() {
		defer close(done)
		for seenCount.Load() < int64(total) {
			v, ok := b.TakeUnsafe()
			if !ok {
				continue
			}
			if v < 1 || v > int64(total) {
				t.Errorf("got out-of-range id %d", v)
				continue
			}
			if seen[v] {
				t.Errorf("id %d observed more than once", v)
			}
			seen[v] = true
			seenCount.Add(1)
		}
	}()

	wg.Wait()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for consumer to observe all %d items (seenCount=%d): "+
			"a lost update under concurrent producers would make this hang forever", total, seenCount.Load())
	}

	if got := seenCount.Load(); got != int64(total) {
		t.Fatalf("seenCount = %d, want %d (lost updates under concurrent producers)", got, total)
	}
	if got := b.Len(); got != 0 {
		t.Fatalf("Len() = %d, want 0 after draining everything", got)
	}
}

func TestRingBufferTakeBatchAndDrain(t *testing.T) {
	b := NewRingBuffer[int](16)
	for i := range 10 {
		b.Add(i)
	}

	batch := b.TakeBatch(4)
	if len(batch) != 4 {
		t.Fatalf("TakeBatch(4) returned %d items, want 4", len(batch))
	}
	for i, v := range batch {
		if v != i {
			t.Errorf("batch[%d] = %d, want %d", i, v, i)
		}
	}

	rest := b.Drain()
	if len(rest) != 6 {
		t.Fatalf("Drain() returned %d items, want 6", len(rest))
	}
	for i, v := range rest {
		if v != i+4 {
			t.Errorf("rest[%d] = %d, want %d", i, v, i+4)
		}
	}
	if !b.Empty() {
		t.Fatal("Empty() = false after draining everything")
	}
}
