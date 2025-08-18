// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"os"
	"path/filepath"
)

func runtimeDir(appslug string) string {
	uid := os.Getuid()

	// Root user: use system-wide directory
	if uid == 0 {
		return filepath.Join("/var/run", appslug)
	}

	// User-specific location
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, "Library", "Application Support", appslug, "Runtime")
	}

	// Fallback to temp
	return filepath.Join("/tmp", appslug)
}
