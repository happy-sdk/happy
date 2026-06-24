// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package app

import (
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

// TestRunAlreadyBootedDoesNotDoubleUnlock is a regression test for a bug
// where Run(), upon detecting the app was already booted, explicitly called
// m.mu.Unlock() and then returned, triggering the deferred m.mu.Unlock()
// installed at the top of Run() a second time. That is a fatal "sync:
// Unlock of unlocked RWMutex" runtime error, not a recoverable panic, so it
// crashes the whole process unconditionally. Run() must be safe to call
// again once the app is already booted.
func TestRunAlreadyBootedDoesNotDoubleUnlock(t *testing.T) {
	m := New(nil)
	m.mu.Lock()
	m.booted = true
	m.mu.Unlock()

	// Must return cleanly without crashing the process.
	m.Run()

	testutils.Assert(t, m.booted, "expected booted to remain true")
}
