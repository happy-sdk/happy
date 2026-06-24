// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package services

import (
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/networking/address"
	"github.com/happy-sdk/happy/sdk/services/service"
	"github.com/happy-sdk/happy/sdk/session"
)

func newTestContainer(t *testing.T) (*Container, *session.Context, func()) {
	t.Helper()

	sess, _, cleanup, err := session.CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	addr, err := address.FromModule("test.local", "container-test")
	if err != nil {
		cleanup()
		t.Fatalf("failed to create address: %v", err)
	}

	svc := New(service.Config{})

	c, err := NewContainer(sess, addr, svc)
	if err != nil {
		cleanup()
		t.Fatalf("failed to create container: %v", err)
	}
	return c, sess, cleanup
}

// TestForceShutdownAlreadyLockedDoesNotPanic is a regression test for a bug
// where ForceShutdown unconditionally called c.mu.Unlock() even when the
// container's mutex was already held by another goroutine (the !IsLocked()
// branch was skipped, so this goroutine never actually locked it). Since
// sync.RWMutex.Unlock does not track ownership, that either panicked with
// "sync: Unlock of unlocked RWMutex" or corrupted another goroutine's
// critical section. ForceShutdown must only unlock the mutex it itself
// locked.
func TestForceShutdownAlreadyLockedDoesNotPanic(t *testing.T) {
	c, sess, cleanup := newTestContainer(t)
	defer cleanup()

	// Simulate another goroutine holding the lock, exactly what IsLocked()
	// is meant to detect.
	c.mu.Lock()
	c.lockInfo.Store("held by another goroutine")

	ready := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		close(ready)
		if err := c.ForceShutdown(sess, nil); err != nil {
			t.Errorf("ForceShutdown returned error: %v", err)
		}
	}()

	<-ready
	// Hold the lock for a moment so ForceShutdown's IsLocked() check, running
	// concurrently, observes it as locked and takes the "already locked"
	// branch -- the one that must not touch a lock it never acquired.
	time.Sleep(10 * time.Millisecond)

	// If ForceShutdown incorrectly released this lock already (the original
	// bug), this panics with "Unlock of unlocked Mutex" instead of cleanly
	// releasing the lock we are still legitimately holding.
	c.mu.Unlock()

	<-done
}

// TestForceShutdownNotLockedDoesNotPanic covers the common case where
// nothing else holds the lock: ForceShutdown should acquire it, release it,
// and proceed to Stop without error.
func TestForceShutdownNotLockedDoesNotPanic(t *testing.T) {
	c, sess, cleanup := newTestContainer(t)
	defer cleanup()

	if err := c.ForceShutdown(sess, nil); err != nil {
		t.Errorf("ForceShutdown returned error: %v", err)
	}
}
