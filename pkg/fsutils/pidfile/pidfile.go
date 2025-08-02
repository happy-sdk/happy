// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package pidfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

var (
	Error = errors.New("pidfile")
)

// File wraps *os.File and provides api for pidfile management.
type File struct {
	*os.File
}

func New(name string, pid int, perm os.FileMode) (*File, error) {
	file, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, perm)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create pidfile: %w", Error, err)
	}

	pf := &File{file}
	if err = pf.Lock(); err != nil {
		_ = pf.Remove()
		return nil, err
	}

	if pid < 1 {
		pid = os.Getpid()
	}

	if pidlen, err := fmt.Fprint(pf, pid); err != nil {
		return nil, fmt.Errorf("%w: set pid: %s", Error, err.Error())
	} else if err = pf.Truncate(int64(pidlen)); err != nil {
		return nil, fmt.Errorf("%w: truncate pidfile: %w", Error, err)
	}

	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("%w: sync pidfile: %w", Error, err)
	}
	return &File{file}, nil
}

func Open(name string) (*File, error) {
	file, err := os.OpenFile(name, os.O_RDONLY, 0640)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to open pidfile: %w", Error, err)
	}
	return &File{file}, nil
}

func (pf *File) PID() (pid int, err error) {
	if _, err = pf.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("%w: failed to seek pidfile: %w", Error, err)
	}
	_, err = fmt.Fscan(pf, &pid)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to read pidfile: %w", Error, err)
	}
	return pid, nil
}

// Lock apply exclusive lock on pidfile.
func (pf *File) Lock() error {
	return lock(pf.Fd())
}

// Unlock remove exclusive lock on an open pidfile.
func (pf *File) Unlock() error {
	return unlock(pf.Fd())
}

func (pf *File) Remove() error {
	defer pf.Close()
	if err := pf.Unlock(); err != nil {
		return err
	}
	return os.Remove(pf.Name())
}

func lock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB)
}

func unlock(fd uintptr) error {
	return syscall.Flock(int(fd), syscall.LOCK_UN)
}
