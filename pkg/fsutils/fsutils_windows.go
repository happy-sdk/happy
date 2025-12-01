// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build windows

package fsutils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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

// fileTimeToTime converts Windows FILETIME to Go time.Time
func fileTimeToTime(ft windows.Filetime) time.Time {
	// FILETIME is 100-nanosecond intervals since January 1, 1601
	// Convert to Unix time (seconds since January 1, 1970)
	nsec := int64(ft.HighDateTime)<<32 + int64(ft.LowDateTime)
	// FILETIME epoch is 1601-01-01, Unix epoch is 1970-01-01
	// Difference: 11644473600 seconds
	unixNsec := nsec - 116444736000000000
	return time.Unix(0, unixNsec*100)
}

func fileStat(f *os.File) (FileInfo, error) {
	handle := windows.Handle(f.Fd())
	var fileInfo windows.ByHandleFileInformation
	err := windows.GetFileInformationByHandle(handle, &fileInfo)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}

	// Get file attributes for mode
	path := f.Name()
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}
	attributes, err := windows.GetFileAttributes(pathPtr)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}

	// Convert Windows file mode to Unix-like mode
	var mode uint16
	if attributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		mode = 0040000 // Directory
	} else {
		mode = 0100000 // Regular file
	}
	if attributes&windows.FILE_ATTRIBUTE_READONLY != 0 {
		mode |= 0444 // Read-only
	} else {
		mode |= 0666 // Read-write
	}

	// Calculate size from high and low DWORDs
	size := uint64(fileInfo.FileSizeHigh)<<32 | uint64(fileInfo.FileSizeLow)

	// Calculate inode from file index high and low
	ino := uint64(fileInfo.FileIndexHigh)<<32 | uint64(fileInfo.FileIndexLow)

	// Windows doesn't have device major/minor in the same way
	// Use volume serial number as a substitute
	devMajor := uint32(fileInfo.VolumeSerialNumber >> 16)
	devMinor := uint32(fileInfo.VolumeSerialNumber & 0xFFFF)

	return FileInfo{
		Name:     filepath.Base(f.Name()),
		Atime:    fileTimeToTime(fileInfo.LastAccessTime),
		Btime:    fileTimeToTime(fileInfo.CreationTime),
		Ctime:    fileTimeToTime(fileInfo.LastWriteTime), // Windows doesn't have separate change time
		Mtime:    fileTimeToTime(fileInfo.LastWriteTime),
		Blksize:  4096, // Default block size on Windows
		Nlink:    uint32(fileInfo.NumberOfLinks),
		Size:     size,
		Blocks:   (size + 511) / 512, // Approximate blocks
		Ino:      ino,
		Mode:     mode,
		Uid:      0, // Windows doesn't have UID
		Gid:      0, // Windows doesn't have GID
		DevMajor: devMajor,
		DevMinor: devMinor,
	}, nil
}

func stat(path string) (FileInfo, error) {
	// Open file to get handle
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}

	handle, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS, // Required for directories
		0,
	)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}
	defer windows.CloseHandle(handle)

	var fileInfo windows.ByHandleFileInformation
	err = windows.GetFileInformationByHandle(handle, &fileInfo)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}

	// Get file attributes for mode
	attributes, err := windows.GetFileAttributes(pathPtr)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}

	// Convert Windows file mode to Unix-like mode
	var mode uint16
	if attributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		mode = 0040000 // Directory
	} else {
		mode = 0100000 // Regular file
	}
	if attributes&windows.FILE_ATTRIBUTE_READONLY != 0 {
		mode |= 0444 // Read-only
	} else {
		mode |= 0666 // Read-write
	}

	// Calculate size from high and low DWORDs
	size := uint64(fileInfo.FileSizeHigh)<<32 | uint64(fileInfo.FileSizeLow)

	// Calculate inode from file index high and low
	ino := uint64(fileInfo.FileIndexHigh)<<32 | uint64(fileInfo.FileIndexLow)

	// Windows doesn't have device major/minor in the same way
	// Use volume serial number as a substitute
	devMajor := uint32(fileInfo.VolumeSerialNumber >> 16)
	devMinor := uint32(fileInfo.VolumeSerialNumber & 0xFFFF)

	return FileInfo{
		Name:     filepath.Base(path),
		Atime:    fileTimeToTime(fileInfo.LastAccessTime),
		Btime:    fileTimeToTime(fileInfo.CreationTime),
		Ctime:    fileTimeToTime(fileInfo.LastWriteTime), // Windows doesn't have separate change time
		Mtime:    fileTimeToTime(fileInfo.LastWriteTime),
		Blksize:  4096, // Default block size on Windows
		Nlink:    uint32(fileInfo.NumberOfLinks),
		Size:     size,
		Blocks:   (size + 511) / 512, // Approximate blocks
		Ino:      ino,
		Mode:     mode,
		Uid:      0, // Windows doesn't have UID
		Gid:      0, // Windows doesn't have GID
		DevMajor: devMajor,
		DevMinor: devMinor,
	}, nil
}

func fileSELinuxContext(_ *os.File) (string, error) {
	// Windows does not support SELinux
	return "", nil
}

func sELinuxContext(_ string) (string, error) {
	// Windows does not support SELinux
	return "", nil
}

func dirBtimeSpan(dir string, recursive bool) (oldest, newest time.Time, bspan time.Duration, err error) {
	first := true

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && path != dir {
				return filepath.SkipDir // Skip subdirectories if not recursive
			}
			return nil // Continue scanning directories
		}

		stat, err := stat(path)
		if err != nil {
			return err
		}

		if first {
			oldest = stat.Btime
			newest = stat.Btime
			first = false
		} else {
			if stat.Btime.Before(oldest) {
				oldest = stat.Btime
			}
			if stat.Btime.After(newest) {
				newest = stat.Btime
			}
		}
		return nil
	})
	if err != nil {
		return oldest, newest, 0, err
	}

	if first {
		return oldest, newest, 0, os.ErrNotExist
	}

	return oldest, newest, newest.Sub(oldest), nil
}
