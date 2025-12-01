// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build darwin

package pidfile

import (
	"syscall"
)

func lock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB)
}

func unlock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_UN)
}
