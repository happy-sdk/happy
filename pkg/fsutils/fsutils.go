// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	Error = errors.New("fsutils")
)

// FileInfo describes a file
// it holds file access, birth, change, and modification times.
type FileInfo struct {
	Name     string
	Atime    time.Time // Last access time
	Btime    time.Time // Birth (creation) time
	Ctime    time.Time // Last status change time
	Mtime    time.Time // Last modification time
	Blksize  uint32
	Nlink    uint32
	Size     uint64
	Blocks   uint64
	Ino      uint64
	Mode     uint16
	Uid      uint32
	Gid      uint32
	DevMajor uint32
	DevMinor uint32
}

// AvailableSpace calculates the available space on the filesystem where path resides.
func AvailableSpace(path string) (uint64, error) {
	return availableSpace(path)
}

// CountFilesAndDirs counts regular files and directories in dir and its subdirectories.
func CountFilesAndDirs(dir string) (filec, dirc int, err error) {
	if _, err := os.Stat(dir); err != nil {
		return 0, 0, fmt.Errorf("%w: can not count files and directories in %s: %s", Error, dir, err.Error())
	}
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

// FileSELinuxContext retrieves the file's SELinux context
// (e.g., "unconfined_u:object_r:config_home_t:s0").
func FileSELinuxContext(f *os.File) (string, error) {
	return fileSELinuxContext(f)
}

// FileStat retrieves rich file statistics for the given file.
func FileStat(file *os.File) (FileInfo, error) {
	return fileStat(file)
}

func IsStdoutStderrFile(expectedPath string) (bool, error) {
	// Stat the expected file
	expectedInfo, err := stat(expectedPath)
	if err != nil {
		return false, fmt.Errorf("%w: failed to stat expected path", Error)
	}

	stdoutStat, err := fileStat(os.Stdout)
	if err != nil {
		return false, fmt.Errorf("%w: failed to stat stdout", Error)
	}

	stderrStat, err := fileStat(os.Stderr)
	if err != nil {
		return false, fmt.Errorf("%w: failed to stat stderr", Error)
	}

	// Compare inode and device
	return expectedInfo.Ino == stdoutStat.Ino &&
		expectedInfo.DevMajor == stdoutStat.DevMajor &&
		expectedInfo.DevMinor == stdoutStat.DevMinor &&
		expectedInfo.Ino == stderrStat.Ino &&
		expectedInfo.DevMajor == stderrStat.DevMajor &&
		expectedInfo.DevMinor == stderrStat.DevMinor, nil
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

// SELinuxContext retrieves the file's SELinux context
// (e.g., "unconfined_u:object_r:config_home_t:s0")
// for the given file path.
func SELinuxContext(path string) (string, error) {
	return sELinuxContext(path)
}

// Stat retrieves rich file statistics for the given file path.
func Stat(name string) (FileInfo, error) {
	return stat(name)
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
