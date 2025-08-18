// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build linux || freebsd || openbsd || netbsd

package fsutils

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
)

func availableSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}

func runtimeDir(appslug string) string {
	uid := os.Getuid()

	// Root user: use system-wide /run directory
	if uid == 0 {
		if runtime.GOOS == "openbsd" {
			return filepath.Join("/var/run", appslug)
		}
		return filepath.Join("/run", appslug)
	}

	// Non-root user: prefer XDG_RUNTIME_DIR
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		return filepath.Join(runtimeDir, appslug)
	}

	// Fallback 1: Standard user runtime directory
	userRuntimeDir := filepath.Join("/run/user", strconv.Itoa(uid), appslug)
	if IsDirectoryAccessible(filepath.Dir(userRuntimeDir)) {
		return userRuntimeDir
	}

	// Fallback 2: Use /tmp if /run/user is not available
	return filepath.Join("/tmp", appslug)
}
