// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// AvailableSpace calculates the available space on the filesystem where path resides.
func AvailableSpace(path string) (uint64, error) {
	return availableSpace(path)
}

// CountFilesAndDirs counts regular files and directories in dir and its subdirectories.
func CountFilesAndDirs(dir string) (filec, dirc int, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirc++
		} else if d.Type().IsRegular() {
			filec++
		}
		return nil
	})
	return filec, dirc, err
}

// DirSize calculates the total size of a directory by traversing it
// and summing the sizes of all encountered files.
// DirSize calculates the total size of regular files in dir and its subdirectories,
// excluding symlinks.
func DirSize(dir string) (int64, error) {
	var size int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// IsDir checks if the given path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsRegular reports whether the path is a regular file.
// Follows symlinks, so a symlink pointing to a regular file returns true.
func IsRegular(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsSymlink checks if the given path is a symbolic link.
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// UserDataDir returns the platform path for shared persistent data.
func UserDataDir(appslug string) string {
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

// UserRuntimeDir return user runtime dir.
func UserRuntimeDir(appslug string) string {
	return runtimeDir(appslug)
}

// UserStateDir returns the platform path for shared state data.
func UserStateDir(appslug string) string {
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
