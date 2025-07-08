// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build linux || darwin || freebsd

package fsutils

import "syscall"

func availableSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}
