// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package rotatefile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/happy-sdk/happy/pkg/fsutils"
)

var (
	Error          = errors.New("rotatefile")
	ErrOption      = fmt.Errorf("%w option", Error)
	ErrMaxSequence = fmt.Errorf("%w maximum sequence %d reached", Error, MaxSequence)
)

const (
	DefaultArchivePerm = 0750
	DefaultFilePerm    = 0640
	MaxSequence        = 99999
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
	archiveDir    string
	archivePerm   fs.FileMode
	filePerm      fs.FileMode
}

func RotateOnOpen() Option {
	return func(rf *File) error {
		rf.rotateOnOpen = true
		return nil
	}
}

// FileMode sets the file mode for the rotated files.
// If mode is 0, DefaultFileMode will be used.
func FileMode(mode fs.FileMode) Option {
	if mode == 0 {
		mode = DefaultFilePerm
	}
	return func(rf *File) error {
		rf.filePerm = mode
		return nil
	}
}

// ArchiveDir sets the directory where rotated files will be archived.
// default is same directory. If directory does not exits it will be created.
// If mode is 0, DefaultArchiveDirMode will be used.
// If oprion is not set file is rotated in same directory.
func ArchiveDir(dir string, perm os.FileMode) Option {
	if perm == 0 {
		perm = DefaultArchivePerm
	}
	return func(rf *File) error {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("%w(archivedir): %s", ErrOption, err.Error())
		}
		if !fsutils.IsDir(abs) {
			if err := os.MkdirAll(abs, perm); err != nil {
				return fmt.Errorf("%w(archivedir): %s", ErrOption, err.Error())
			}
		}
		rf.archiveDir = abs
		rf.archivePerm = perm
		return nil
	}
}

func RotatedFilePrefix(prefix string) Option {
	return func(rf *File) error {
		rf.rotatedPrefix = prefix
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
		name:        filepath.Base(abspath),
		dir:         dir,
		abspath:     abspath,
		archivePerm: DefaultArchivePerm,
		filePerm:    DefaultFilePerm,
	}

	for _, opt := range opts {
		if err := opt(file); err != nil {
			return nil, err
		}
	}

	if _, err := file.open(); err != nil {
		return nil, fmt.Errorf("%w: failed to open: %w", Error, err)
	}

	if file.rotateOnOpen {
		if err := file.rotate(true); err != nil {
			return nil, err
		}
	}
	return file, nil
}

func (rf *File) Write(p []byte) (n int, err error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.file == nil {
		return 0, fmt.Errorf("%w: write on closed file", Error)
	}
	return rf.file.Write(p)
}

func (rf *File) Close() error {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	file := rf.file
	if file != nil {
		rf.file = nil
		return file.Close()
	}
	return nil
}

// File calls os.OpenFile with given flags.
// It returns a new file descriptor that shares the same underlying file,
// but does not replace internal file descriptor.
// Caller is responsible for closing the returned file descriptor.
func (rf *File) OpenFile(flag int) (*os.File, error) {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return os.OpenFile(rf.abspath, flag, rf.filePerm)
}

func (rf *File) Name() string {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.name
}

func (rf *File) Dir() string {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.dir
}

func (rf *File) Rotations() int {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.rotations
}

func (rf *File) PreviousRotation() time.Time {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.prevRotation
}

// Stat retrieves the file's stats using unix.Statx.
func (rf *File) Stat() (fsutils.FileInfo, error) {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.stat()
}

func (rf *File) Rotate() (err error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.rotate(false)
}

func (rf *File) rotate(force bool) (err error) {

	// if file is current file is empty then skip and is not internal force call.
	if !force {
		stat, err := fsutils.Stat(filepath.Join(rf.dir, rf.name))
		if err != nil || stat.Size == 0 {
			return nil
		}
	}

	// Generate rotated filename based on file's last modification time
	rotatedPath := rf.generateRotatedFileName(rf.btime)
	rotatedPath, err = rf.findNextSequenceFilePath(rotatedPath)
	if err != nil {
		if errors.Is(err, ErrMaxSequence) {
			return ErrMaxSequence
		}
		return fmt.Errorf("%w: failed to find next sequence file path: %w", Error, err)
	}

	currentFile := filepath.Join(rf.dir, rf.name)

	if rf.file != nil {
		rf.prevRotation = rf.btime
		if err := rf.file.Close(); err != nil {
			return fmt.Errorf("%w: failed to close file: %w", Error, err)
		}
		rf.file = nil
	}

	if rf.archiveDir != "" && !fsutils.IsDir(rf.archiveDir) {
		if err := os.MkdirAll(rf.archiveDir, rf.archivePerm); err != nil {
			return fmt.Errorf("%w: failed to create archive directory for %s: %w", Error, rf.name, err)
		}
	}

	if err := os.Rename(currentFile, rotatedPath); err != nil {
		return fmt.Errorf("%w: failed to rotate file: %w", Error, err)
	}

	if _, err := rf.open(); err != nil {
		return err
	}

	rf.rotations++
	return nil
}

// SELinuxContext retrieves the file's SELinux context
// (e.g., "unconfined_u:object_r:config_home_t:s0").
func (rf *File) SELinuxContext() (string, error) {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	if rf.file == nil {
		return "", errors.New("file not open")
	}
	v, err := fsutils.FileSELinuxContext(rf.file)
	if err != nil {
		return "", fmt.Errorf("%w: failed to retrieve SELinux context: %w", Error, err)
	}
	return v, nil
}

func (rf *File) open() (*File, error) {
	if rf.file != nil {
		if err := rf.file.Close(); err != nil {
			return nil, fmt.Errorf("%w: failed to close previously opened file: %w", Error, err)
		}
	}

	file, err := os.OpenFile(rf.abspath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, rf.filePerm)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to open file: %w", Error, err)
	}

	rf.file = file
	stat, err := rf.stat()
	if err != nil {
		return nil, err
	}
	rf.btime = stat.Btime

	// check last rotation
	if rf.prevRotation.IsZero() {
		rotatedPath := rf.generateRotatedFileName(rf.btime)
		last, rotations, err := rf.findLastRotatedFilePath(rotatedPath)
		if err != nil {
			return nil, err
		}
		if rstat, err := fsutils.Stat(last); err == nil {
			rf.prevRotation = rstat.Btime
			rf.rotations = rotations
		} else {
			rf.rotations = 1
			rf.prevRotation = stat.Btime
		}
	}
	return rf, nil
}

// stat is internal to get platform independent file information.
// It is sets Btime closest for current platform.
func (rf *File) stat() (fsutils.FileInfo, error) {
	if rf.file == nil {
		return fsutils.FileInfo{}, fmt.Errorf("%w: can not stat file not open", Error)
	}
	return fsutils.FileStat(rf.file)
}

// generateRotatedFileName creates the rotated file path based on the file's timestamp
func (f *File) generateRotatedFileName(timestamp time.Time) string {
	dir := f.dir
	if f.archiveDir != "" {
		dir = f.archiveDir
	}
	base := filepath.Base(f.name)
	ext := filepath.Ext(base)

	dateStr := timestamp.Format("20060102")
	return filepath.Join(dir, fmt.Sprintf("%s%s%s", f.rotatedPrefix, dateStr, ext))
}

// findNextSequenceFilePath finds the next available sequence file path number for rotation file
// Uses zero-padded 5-digit sequence numbers for proper lexicographic sorting
func (f *File) findNextSequenceFilePath(fpath string) (string, error) {
	baseName := filepath.Base(fpath)
	dir := filepath.Dir(fpath)
	if f.archiveDir != "" {
		dir = f.archiveDir
	}
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Use shared helper to find max sequence
	maxSequence := findMaxSequenceInDir(dir, nameWithoutExt, ext)

	// Determine next sequence
	nextSequence := maxSequence + 1

	// If no sequences found, check if base file exists
	if maxSequence == 0 {
		if _, err := os.Stat(fpath); err != nil {
			// Base file doesn't exist, return it as-is
			return fpath, nil
		}
		// Base file exists, start with sequence 1
		nextSequence = 1
	}

	var err error
	if nextSequence > MaxSequence {
		err = ErrMaxSequence
	}

	// Return next sequence with zero-padding (5 digits)
	return filepath.Join(dir, fmt.Sprintf("%s.%05d%s", nameWithoutExt, nextSequence, ext)), err
}

// findLastRotatedFilePath finds the last rotated file path and the number of rotations
func (rf *File) findLastRotatedFilePath(seqpath string) (last string, rotations int, err error) {
	baseName := filepath.Base(seqpath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	dir := rf.dir
	if rf.archiveDir != "" {
		dir = rf.archiveDir
	}

	maxSequence := findMaxSequenceInDir(dir, nameWithoutExt, ext)

	if maxSequence == 0 {
		return seqpath, 1, nil
	}

	lastFile := filepath.Join(dir, fmt.Sprintf("%s.%05d%s", nameWithoutExt, maxSequence, ext))
	return lastFile, maxSequence + 1, nil
}

// findMaxSequenceInDir is a shared helper that finds the highest sequence number
// in a directory for files matching the pattern nameWithoutExt.XXXXX.ext
// Made standalone so it can be used by both methods and standalone functions
func findMaxSequenceInDir(dir, nameWithoutExt, ext string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	maxSequence := 0
	expectedPrefix := nameWithoutExt + "."

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Must match our pattern: nameWithoutExt.XXXXX.ext
		if !strings.HasPrefix(name, expectedPrefix) || !strings.HasSuffix(name, ext) {
			continue
		}

		middle := strings.TrimPrefix(strings.TrimSuffix(name, ext), expectedPrefix)

		if !isAllDigits(middle) {
			continue
		}

		if seq, err := strconv.Atoi(middle); err == nil && seq > 0 && seq > maxSequence {
			maxSequence = seq
		}
	}

	return maxSequence
}

// isAllDigits checks if a string contains only digits
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
