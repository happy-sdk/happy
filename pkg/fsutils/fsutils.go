// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"os"
	"path/filepath"
	"runtime"
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

// DataDir returns the platform path for shared persistent data.
func DataDir(appslug string) string {
	switch runtime.GOOS {
	case "linux":
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
			return filepath.Join(dataHome, appslug)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", appslug)
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", appslug)
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, appslug)
		}
		return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", appslug)
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "."+appslug, "data")
	}
}

// StateDir returns the platform path for shared state data.
func StateDir(appslug string) string {
	switch runtime.GOOS {
	case "linux":
		if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
			return filepath.Join(stateHome, appslug)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "state", appslug)
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", appslug, "State")
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, appslug, "State")
		}
		return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", appslug, "State")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "."+appslug, "state")
	}
}
