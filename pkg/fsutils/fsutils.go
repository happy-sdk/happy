// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"os"
	"path/filepath"
)

// DirSize calculates the total size of a directory by traversing it
// and summing the sizes of all encountered files.
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func AvailableSpace(path string) (uint64, error) {
	return availableSpace(path)
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsRegular(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func RuntimeDir(appslug string) string {
	return runtimeDir(appslug)
}

func IsDirectoryAccessible(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
