// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

//go:build linux || freebsd || openbsd || netbsd

package fsutils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
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
	if IsDir(filepath.Dir(userRuntimeDir)) {
		return userRuntimeDir
	}

	// Fallback 2: Use /tmp if /run/user is not available
	return filepath.Join("/tmp", appslug)
}

func fileStat(f *os.File) (FileInfo, error) {
	sc, err := f.SyscallConn()
	if err != nil {
		return FileInfo{}, err
	}

	var statx unix.Statx_t
	var statxErr error
	err = sc.Control(func(fd uintptr) {
		statxErr = unix.Statx(
			int(fd),
			"",
			unix.AT_EMPTY_PATH|unix.AT_STATX_SYNC_AS_STAT,
			unix.STATX_ALL,
			&statx,
		)
	})
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}
	if statxErr != nil {
		return FileInfo{}, fmt.Errorf("%w: statx %s", Error, statxErr.Error())
	}

	return FileInfo{
		Atime:   time.Unix(int64(statx.Atime.Sec), int64(statx.Atime.Nsec)),
		Btime:   time.Unix(int64(statx.Btime.Sec), int64(statx.Btime.Nsec)),
		Ctime:   time.Unix(int64(statx.Ctime.Sec), int64(statx.Ctime.Nsec)),
		Mtime:   time.Unix(int64(statx.Mtime.Sec), int64(statx.Mtime.Nsec)),
		Blksize: statx.Blksize,
		Nlink:   statx.Nlink,
		Size:    statx.Size,
		Blocks:  statx.Blocks,
		Ino:     statx.Ino,
		Mode:    statx.Mode,
		Uid:     statx.Uid,
		Gid:     statx.Gid,
	}, nil
}

func stat(path string) (FileInfo, error) {
	var statx unix.Statx_t
	err := unix.Statx(
		unix.AT_FDCWD,
		path,
		unix.AT_STATX_SYNC_AS_STAT,
		unix.STATX_ALL,
		&statx,
	)
	if err != nil {
		return FileInfo{}, fmt.Errorf("%w: %s", Error, err.Error())
	}

	return FileInfo{
		Atime:   time.Unix(int64(statx.Atime.Sec), int64(statx.Atime.Nsec)),
		Btime:   time.Unix(int64(statx.Btime.Sec), int64(statx.Btime.Nsec)),
		Ctime:   time.Unix(int64(statx.Ctime.Sec), int64(statx.Ctime.Nsec)),
		Mtime:   time.Unix(int64(statx.Mtime.Sec), int64(statx.Mtime.Nsec)),
		Blksize: statx.Blksize,
		Nlink:   statx.Nlink,
		Size:    statx.Size,
		Blocks:  statx.Blocks,
		Ino:     statx.Ino,
		Mode:    statx.Mode,
		Uid:     statx.Uid,
		Gid:     statx.Gid,
	}, nil
}

func fileSELinuxContext(f *os.File) (string, error) {
	sc, err := f.SyscallConn()
	if err != nil {
		return "", err
	}

	buf := make([]byte, 256)
	var n int
	var xattrErr error

	err = sc.Control(func(fd uintptr) {
		n, xattrErr = unix.Fgetxattr(int(fd), "security.selinux", buf)
	})
	if err != nil {
		return "", err
	}
	if xattrErr != nil {
		if errors.Is(xattrErr, unix.ENODATA) || errors.Is(xattrErr, unix.ENOTSUP) {
			return "", nil // No SELinux context available
		}
		return "", xattrErr
	}

	// Trim null terminator from the SELinux context
	return strings.TrimRight(string(buf[:n]), "\x00"), nil
}

func sELinuxContext(path string) (string, error) {
	buf := make([]byte, 256)
	n, err := unix.Getxattr(path, "security.selinux", buf)
	if err != nil {
		if errors.Is(err, unix.ENODATA) || errors.Is(err, unix.ENOTSUP) {
			return "", nil // No SELinux context available
		}
		return "", err
	}

	// Trim null terminator from the SELinux context
	return strings.TrimRight(string(buf[:n]), "\x00"), nil
}

func dirBtimeSpan(dir string, recursive bool) (oldest, newest time.Time, bspan time.Duration, err error) {
	var first bool = true

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
