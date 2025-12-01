// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package pidfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	pidfilePath := filepath.Join(tmpDir, "test.pid")

	// Test creating a new pidfile with default PID (current process)
	pf, err := New(pidfilePath, 0, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile")
	testutils.NotNil(t, pf, "expected non-nil pidfile")
	defer func() { _ = pf.Close() }()

	// Verify file exists
	_, err = os.Stat(pidfilePath)
	testutils.NoError(t, err, "expected pidfile to exist")

	// Test creating with explicit PID
	pidfilePath2 := filepath.Join(tmpDir, "test2.pid")
	pf2, err := New(pidfilePath2, 12345, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile with explicit PID")
	testutils.NotNil(t, pf2, "expected non-nil pidfile")
	defer func() { _ = pf2.Close() }()

	// Verify PID was written correctly
	pid, err := pf2.PID()
	testutils.NoError(t, err, "expected no error reading PID")
	testutils.Equal(t, 12345, pid, "expected PID to match")

	// Test creating pidfile that already exists (should fail)
	_, err = New(pidfilePath, 0, 0644)
	testutils.Error(t, err, "expected error when pidfile already exists")
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	pidfilePath := filepath.Join(tmpDir, "test.pid")

	// Create a pidfile first
	pf, err := New(pidfilePath, 12345, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile")
	_ = pf.Close()

	// Test opening existing pidfile
	pf2, err := Open(pidfilePath)
	testutils.NoError(t, err, "expected no error opening pidfile")
	testutils.NotNil(t, pf2, "expected non-nil pidfile")
	defer func() { _ = pf2.Close() }()

	// Verify we can read the PID
	pid, err := pf2.PID()
	testutils.NoError(t, err, "expected no error reading PID")
	testutils.Equal(t, 12345, pid, "expected PID to match")

	// Test opening non-existent pidfile
	_, err = Open(filepath.Join(tmpDir, "nonexistent.pid"))
	testutils.Error(t, err, "expected error opening non-existent pidfile")
}

func TestPID(t *testing.T) {
	tmpDir := t.TempDir()
	pidfilePath := filepath.Join(tmpDir, "test.pid")

	// Create pidfile with known PID
	pf, err := New(pidfilePath, 99999, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile")
	defer func() { _ = pf.Close() }()

	// Read PID
	pid, err := pf.PID()
	testutils.NoError(t, err, "expected no error reading PID")
	testutils.Equal(t, 99999, pid, "expected PID to match")

	// Read again (should work multiple times)
	pid2, err := pf.PID()
	testutils.NoError(t, err, "expected no error reading PID again")
	testutils.Equal(t, 99999, pid2, "expected PID to match on second read")
}

func TestLock(t *testing.T) {
	tmpDir := t.TempDir()
	pidfilePath := filepath.Join(tmpDir, "test.pid")

	// Create and lock pidfile
	pf, err := New(pidfilePath, 0, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile")
	defer func() { _ = pf.Close() }()

	err = pf.Lock()
	_ = err

	// Test that we can't create another pidfile with the same path
	_, err = New(pidfilePath, 0, 0644)
	testutils.Error(t, err, "expected error when trying to create locked pidfile")
}

func TestUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	pidfilePath := filepath.Join(tmpDir, "test.pid")

	// Create pidfile
	pf, err := New(pidfilePath, 0, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile")

	// Unlock should succeed
	err = pf.Unlock()
	testutils.NoError(t, err, "expected no error unlocking pidfile")

	// Close the file
	err = pf.Close()
	testutils.NoError(t, err, "expected no error closing pidfile")
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	pidfilePath := filepath.Join(tmpDir, "test.pid")

	// Create pidfile
	pf, err := New(pidfilePath, 0, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile")

	// Verify file exists
	_, err = os.Stat(pidfilePath)
	testutils.NoError(t, err, "expected pidfile to exist")

	// Remove should succeed
	err = pf.Remove()
	testutils.NoError(t, err, "expected no error removing pidfile")

	// Verify file is removed
	_, err = os.Stat(pidfilePath)
	testutils.Error(t, err, "expected error when statting removed pidfile")
	testutils.Assert(t, os.IsNotExist(err), "expected IsNotExist error")

}

func TestFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	pidfilePath := filepath.Join(tmpDir, "test.pid")

	pf, err := New(pidfilePath, 0, 0644)
	testutils.NoError(t, err, "expected no error creating pidfile")
	defer func() { _ = pf.Close() }()

	name := pf.Name()
	testutils.Equal(t, pidfilePath, name, "expected filename to match")

	testutils.Assert(t, pf.File != nil, "expected embedded File to be non-nil")
}
