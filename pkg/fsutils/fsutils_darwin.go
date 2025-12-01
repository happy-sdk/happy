// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build darwin

package fsutils

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
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

func availableSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}

func fileStat(f *os.File) (FileInfo, error) {
	sc, err := f.SyscallConn()
	if err != nil {
		return FileInfo{}, err
	}

	var stat unix.Stat_t
	var statErr error
	err = sc.Control(func(fd uintptr) {
		statErr = unix.Fstat(int(fd), &stat)
	})
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}
	if statErr != nil {
		return FileInfo{}, fmt.Errorf("%w: fstat %s", Error, statErr.Error())
	}

	// On Darwin, birth time (Btime) is not directly available in Stat_t
	// We'll use Ctime as a fallback, or try to get it from extended attributes
	btime := time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)

	return FileInfo{
		Name:     filepath.Base(f.Name()),
		Atime:    time.Unix(stat.Atim.Sec, stat.Atim.Nsec),
		Btime:    btime, // Birth time not directly available, using Ctime as fallback
		Ctime:    time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec),
		Mtime:    time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec),
		Blksize:  uint32(stat.Blksize),
		Nlink:    uint32(stat.Nlink),
		Size:     uint64(stat.Size),
		Blocks:   uint64(stat.Blocks),
		Ino:      stat.Ino,
		Mode:     uint16(stat.Mode),
		Uid:      stat.Uid,
		Gid:      stat.Gid,
		DevMajor: unix.Major(uint64(stat.Dev)),
		DevMinor: unix.Minor(uint64(stat.Dev)),
	}, nil
}

func stat(path string) (FileInfo, error) {
	var stat unix.Stat_t
	err := unix.Stat(path, &stat)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}

	// On Darwin, birth time (Btime) is not directly available in Stat_t
	// We'll use Ctime as a fallback, or try to get it from extended attributes
	btime := time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)

	return FileInfo{
		Name:     filepath.Base(path),
		Atime:    time.Unix(stat.Atim.Sec, stat.Atim.Nsec),
		Btime:    btime, // Birth time not directly available, using Ctime as fallback
		Ctime:    time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec),
		Mtime:    time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec),
		Blksize:  uint32(stat.Blksize),
		Nlink:    uint32(stat.Nlink),
		Size:     uint64(stat.Size),
		Blocks:   uint64(stat.Blocks),
		Ino:      stat.Ino,
		Mode:     uint16(stat.Mode),
		Uid:      stat.Uid,
		Gid:      stat.Gid,
		DevMajor: unix.Major(uint64(stat.Dev)),
		DevMinor: unix.Minor(uint64(stat.Dev)),
	}, nil
}

func fileSELinuxContext(_ *os.File) (string, error) {
	// Darwin (macOS) does not support SELinux
	return "", nil
}

func sELinuxContext(_ string) (string, error) {
	// Darwin (macOS) does not support SELinux
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
