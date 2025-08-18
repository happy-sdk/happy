// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func availableSpace(path string) (uint64, error) {
	lpFreeBytesAvailable := uint64(0)
	lpTotalNumberOfBytes := uint64(0)
	lpTotalNumberOfFreeBytes := uint64(0)

	drive := windows.StringToUTF16Ptr(path)

	err := windows.GetDiskFreeSpaceEx(
		drive,
		&lpFreeBytesAvailable,
		&lpTotalNumberOfBytes,
		&lpTotalNumberOfFreeBytes,
	)
	return lpFreeBytesAvailable, err
}

func runtimeDir(appslug string) string {
	if isWindowsAdmin() {
		// System-wide location for admin/service
		if programData := os.Getenv("PROGRAMDATA"); programData != "" {
			return filepath.Join(programData, appslug)
		}
		return filepath.Join("C:", "ProgramData", appslug)
	}

	// User-specific location
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		return filepath.Join(localAppData, appslug, "Runtime")
	}

	// Fallback to temp
	if temp := os.Getenv("TEMP"); temp != "" {
		return filepath.Join(temp, appslug)
	}
	return filepath.Join("C:", "temp", appslug)
}

// isWindowsAdmin checks if running with administrator privileges on Windows
// Simple heuristic: check if we can write to a system directory
func isWindowsAdmin() bool {
	testPath := filepath.Join(os.Getenv("SYSTEMROOT"), "temp", "admin_test")
	file, err := os.Create(testPath)
	if err != nil {
		return false
	}
	file.Close()
	_ = os.Remove(testPath)
	return true
}
