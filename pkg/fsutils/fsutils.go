// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	Error = errors.New("fsutils")
)

// FileInfo describes a file
// it holds file access, birth, change, and modification times.
type FileInfo struct {
	Atime   time.Time // Last access time
	Btime   time.Time // Birth (creation) time
	Ctime   time.Time // Last status change time
	Mtime   time.Time // Last modification time
	Blksize uint32
	Nlink   uint32
	Size    uint64
	Blocks  uint64
	Ino     uint64
	Mode    uint16
	Uid     uint32
	Gid     uint32
}

// AvailableSpace calculates the available space on the filesystem where path resides.
func AvailableSpace(path string) (uint64, error) {
	return availableSpace(path)
}

// CompressDir compresses the specified directory into a .tar.gz archive.
// The output file is writen to file specified by tarpath.
func CompressDir(dir, tarpath string) error {
	if _, err := os.Stat(tarpath); err == nil {
		return fmt.Errorf("%w: tarpath exists", Error)
	}
	// Ensure the directory exists
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("%w: src dir does not exist: %s", Error, err.Error())
	}

	tarfile, err := os.OpenFile(tarpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("%w: failed to create destination file: %s", Error, err.Error())
	}
	defer tarfile.Close()

	gw := gzip.NewWriter(tarfile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		header.Name = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%w: %s", Error, err)
	}
	return nil
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

// DirBtimeSpan returns the time duration between the oldest and newest file
// in the directory based on their Btime (creation time). If recursive is true,
// it includes files in subdirectories; otherwise, it scans only the top-level directory.
func DirBtimeSpan(dir string, recursive bool) (oldest, newest time.Time, bspan time.Duration, err error) {
	return dirBtimeSpan(dir, recursive)
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

// FileSELinuxContext retrieves the file's SELinux context
// (e.g., "unconfined_u:object_r:config_home_t:s0").
func FileSELinuxContext(f *os.File) (string, error) {
	return fileSELinuxContext(f)
}

// FileStat retrieves rich file statistics for the given file.
func FileStat(file *os.File) (FileInfo, error) {
	return fileStat(file)
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
