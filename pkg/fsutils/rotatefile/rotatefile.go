// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package rotatefile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/fsutils"
)

type Option func(*File) error

type File struct {
	mu            sync.RWMutex
	name          string
	dir           string
	abspath       string
	file          *os.File
	rotations     int
	prevRotation  time.Time
	btime         time.Time
	rotateOnOpen  bool
	rotatedPrefix string
}

func RotateOnOpen() Option {
	return func(f *File) error {
		f.rotateOnOpen = true
		return nil
	}
}

func RotatedFilePrefix(prefix string) Option {
	return func(f *File) error {
		f.rotatedPrefix = prefix
		return nil
	}
}

func Open(name string, opts ...Option) (*File, error) {
	abspath, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(abspath)
	file := &File{
		name:    filepath.Base(abspath),
		dir:     dir,
		abspath: abspath,
	}

	for _, opt := range opts {
		if err := opt(file); err != nil {
			return nil, err
		}
	}

	if _, err := file.open(); err != nil {
		return nil, err
	}

	if file.rotateOnOpen {
		if err := file.Rotate(); err != nil {
			return nil, err
		}
	}
	return file, nil
}

func (l *File) Write(p []byte) (n int, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.file == nil {
		return 0, errors.New("file not open")
	}
	return l.file.Write(p)
}

func (l *File) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// File calls os.OpenFile with given flags and permissions.
func (l *File) OpenFile(flag int, perm os.FileMode) (*os.File, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return os.OpenFile(l.abspath, flag, perm)
}

func (l *File) Name() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.name
}

func (l *File) Dir() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.dir
}

func (l *File) Rotations() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.rotations
}

func (l *File) PreviousRotation() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.prevRotation
}

// Stat retrieves the file's stats using unix.Statx.
func (f *File) Stat() (fsutils.FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stat()
}

func (f *File) Rotate() (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// if file is current file is empty then skip
	if stat, err := os.Stat(filepath.Join(f.dir, f.name)); err != nil {
		return nil
	} else if stat.Size() == 0 {
		return nil
	}

	// Generate rotated filename based on file's last modification time
	rotatedPath := f.generateRotatedFileName(f.btime)
	rotatedPath = f.findNextSequenceFilePath(rotatedPath)

	currentFile := filepath.Join(f.dir, f.name)

	if f.file != nil {
		f.prevRotation = f.btime
		if err = f.file.Close(); err != nil {
			return err
		}
	}

	if err := os.Rename(
		currentFile,
		rotatedPath); err != nil {
		return fmt.Errorf("failed to rotate file: %w", err)
	}

	if _, err := f.open(); err != nil {
		return err
	}

	f.rotations++

	return nil
}

// SELinuxContext retrieves the file's SELinux context
// (e.g., "unconfined_u:object_r:config_home_t:s0").
func (f *File) SELinuxContext() (string, error) {
	return fsutils.FileSELinuxContext(f.file)
}

func (f *File) open() (*File, error) {
	file, err := os.OpenFile(f.abspath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return nil, err
	}
	f.file = file
	stat, err := f.stat()
	if err != nil {
		return nil, err
	}
	f.btime = stat.Btime

	// check last rotation
	if f.prevRotation.IsZero() {
		rotatedPath := f.generateRotatedFileName(f.btime)
		last, rotations, err := f.findLastRotatedFilePath(rotatedPath)
		if err != nil {
			return nil, err
		}
		if stat, err := fsutils.Stat(last); err == nil {
			f.prevRotation = stat.Btime
			f.rotations = rotations
		}
	}
	return f, nil
}

func (f *File) stat() (fsutils.FileInfo, error) {
	return fsutils.FileStat(f.file)
}

// generateRotatedFileName creates the rotated file path based on the file's timestamp
func (f *File) generateRotatedFileName(timestamp time.Time) string {
	dir := f.dir
	base := filepath.Base(f.name)
	ext := filepath.Ext(base)

	dateStr := timestamp.Format("20060102")
	return filepath.Join(dir, fmt.Sprintf("%s%s%s", f.rotatedPrefix, dateStr, ext))
}

// findNextSequenceFilePath finds the next available sequence file path number for rotation file
func (f *File) findNextSequenceFilePath(fpath string) string {
	baseName := filepath.Base(fpath)
	dir := filepath.Dir(fpath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Find existing files with sequence numbers
	entries, err := os.ReadDir(dir)
	if err != nil {
		// just use .1
		return filepath.Join(dir, fmt.Sprintf("%s.1%s", nameWithoutExt, ext))
	}

	maxSequence := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || name == baseName {
			continue
		}
		// Check if file matches our pattern (e.g., app-20060102.1.log)
		if strings.HasPrefix(name, nameWithoutExt+".") && strings.HasSuffix(name, ext) {
			middle := strings.TrimPrefix(strings.TrimSuffix(name, ext), nameWithoutExt+".")
			if seq, err := strconv.Atoi(middle); err == nil && seq > maxSequence {
				maxSequence = seq
			}
		}
	}

	if maxSequence == 0 {
		if _, err := os.Stat(fpath); err != nil {
			return fpath
		}
	}

	return filepath.Join(dir, fmt.Sprintf("%s.%d%s", nameWithoutExt, maxSequence+1, ext))
}

// findLastRotatedFilePath finds the last rotated file path and the number of rotations
func (f *File) findLastRotatedFilePath(seqpath string) (last string, rotations int, err error) {

	baseName := filepath.Base(seqpath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Find existing files with sequence numbers
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return seqpath, 0, nil
	}

	maxSequence := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check if file matches our pattern (e.g., app-20060102.1.log)
		if strings.HasPrefix(name, nameWithoutExt+".") && strings.HasSuffix(name, ext) {
			middle := strings.TrimPrefix(strings.TrimSuffix(name, ext), nameWithoutExt+".")
			if seq, err := strconv.Atoi(middle); err == nil && seq > maxSequence {
				maxSequence = seq
			}
		}
	}

	if maxSequence == 0 {
		return seqpath, 1, nil
	}

	return filepath.Join(f.dir, fmt.Sprintf("%s.%d%s", nameWithoutExt, maxSequence, ext)), maxSequence + 1, nil
}
