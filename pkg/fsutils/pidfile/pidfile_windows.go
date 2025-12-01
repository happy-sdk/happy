// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build windows

package pidfile

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modkernel32    = windows.NewLazySystemDLL("kernel32.dll")
	procLockFileEx = modkernel32.NewProc("LockFileEx")
	procUnlockFile = modkernel32.NewProc("UnlockFile")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 0x2
	LOCKFILE_FAIL_IMMEDIATELY = 0x1
)

func lock(fd uintptr) error {
	var overlapped windows.Overlapped
	// LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY
	ret, _, _ := procLockFileEx.Call(
		fd,
		LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY,
		0,
		0,
		0xFFFFFFFF,
		0xFFFFFFFF,
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if ret == 0 {
		// Lock failed - check if it's because file is already locked
		errno := windows.GetLastError()
		if errno == windows.ERROR_LOCK_VIOLATION {
			return windows.ERROR_LOCK_VIOLATION
		}
		return errno
	}
	return nil
}

func unlock(fd uintptr) error {
	ret, _, _ := procUnlockFile.Call(
		fd,
		0,
		0,
		0xFFFFFFFF,
		0xFFFFFFFF,
	)
	if ret == 0 {
		return windows.GetLastError()
	}
	return nil
}
