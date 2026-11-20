// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package services

import (
	"errors"
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

// TestContainerStart is a regression test for dead code in Start: an
// `if err != nil` check against the named return `err` could never fire,
// since every preceding error path returned its own shadowed local `err`
// directly (returning before that check is ever reached) -- this exercises
// the happy path through to its end to confirm removing that dead block
// didn't change behavior.
func TestContainerStart(t *testing.T) {
	c, sess, cleanup := newTestContainer(t)
	defer cleanup()

	if err := c.Start(sess.Context(), sess); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if !c.Info().Running() {
		t.Error("expected service to be running after Start")
	}
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

// TestServiceIsNotRunningUntilItsStartSucceeded is a regression test: Start
// marked the service running before its start action ran, so a loader polling
// the service while the action was slow saw it running, finished without an
// error, and the failure that followed went unnoticed.
func TestServiceIsNotRunningUntilItsStartSucceeded(t *testing.T) {
	sess, _, cleanup, err := session.CreateTestSession(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	addr, err := address.FromModule("test.local", "slow-start")
	if err != nil {
		t.Fatal(err)
	}

	entered, release := make(chan struct{}), make(chan struct{})
	svc := New(service.Config{})
	svc.OnStart(func(*session.Context) error {
		close(entered)
		<-release
		return errors.New("port taken")
	})
	c, err := NewContainer(sess, addr, svc)
	if err != nil {
		t.Fatal(err)
	}

	// Info() takes the container lock Start holds; the loader reads the same
	// pointer through the session.
	info := c.Info()
	done := make(chan error, 1)
	go func() { done <- c.Start(sess.Context(), sess) }()
	<-entered
	if info.Running() {
		t.Error("running while the start action is still in progress")
	}
	if !info.Starting() {
		t.Error("not starting while the start action is in progress")
	}
	close(release)
	if err := <-done; err == nil {
		t.Fatal("Start should return the start action's error")
	}
	if info.Running() || info.Starting() || !info.Failed() {
		t.Errorf("after a failed start: running=%v starting=%v failed=%v", info.Running(), info.Starting(), info.Failed())
	}
}
