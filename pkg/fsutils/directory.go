// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package fsutils

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

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

// BackupDir archives srcDir into outArchive as a tar.gz file and optionally deletes source files.
// Skips files for which skip(path) returns true. If removeSrc is true and srcDir is empty after
// removal (e.g., due to skipped paths), srcDir is also removed. All errors are wrapped with Error.

// IsDir checks if the given path is a directory.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func BackupDir(ctx context.Context, srcDir, outArchive string, removeSrc bool, skip func(string) bool) (err error) {
	// Normalize paths to avoid comparison issues
	srcDir = filepath.Clean(srcDir)
	outArchive = filepath.Clean(outArchive)

	// Check if output archive exists
	if _, err := os.Stat(outArchive); err == nil {
		return fmt.Errorf("%w: destination archive %s already exists", Error, outArchive)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("%w: stat destination %s: %w", Error, outArchive, err)
	}

	// Ensure source directory exists and is a directory
	srcInfo, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("%w: source directory %s: %w", Error, srcDir, err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%w: source %s is not a directory", Error, srcDir)
	}

	// Check if output archive would be inside source directory
	if strings.HasPrefix(outArchive, srcDir+string(filepath.Separator)) {
		return fmt.Errorf("%w: output archive %s cannot be inside source directory %s", Error, outArchive, srcDir)
	}

	// Create output file with atomic write (write to temp, then rename)
	tempArchive := outArchive + ".tmp"
	out, err := os.OpenFile(tempArchive, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("%w: create temporary archive %s: %w", Error, tempArchive, err)
	}

	// Set up gzip and tar writers
	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)

	archiveSuccess := false

	// Defer cleanup with proper error handling
	defer func() {
		// Close writers in correct order
		if e := tw.Close(); e != nil && err == nil {
			err = fmt.Errorf("%w: close tar writer: %w", Error, e)
		}
		if e := gw.Close(); e != nil && err == nil {
			err = fmt.Errorf("%w: close gzip writer: %w", Error, e)
		}
		if e := out.Close(); e != nil && err == nil {
			err = fmt.Errorf("%w: close archive file: %w", Error, e)
		}

		// Handle temp file cleanup and atomic rename
		if archiveSuccess && err == nil {
			if e := os.Rename(tempArchive, outArchive); e != nil {
				err = fmt.Errorf("%w: rename temp archive: %w", Error, e)
				os.Remove(tempArchive) // Clean up temp file on rename failure
			}
		} else {
			os.Remove(tempArchive) // Clean up temp file on failure
		}
	}()

	// Collect all files first to ensure consistent view
	var filesToArchive []string
	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%w: walk %s: %w", Error, path, err)
		}

		// Skip root directory, temp archive, and user-specified files
		if path == srcDir || path == tempArchive || path == outArchive || (skip != nil && skip(path)) {
			return nil
		}

		filesToArchive = append(filesToArchive, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("%w: collect files to archive: %w", Error, err)
	}

	// Archive files with controlled concurrency - don't use errgroup context derivation
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Limit concurrency
	errChan := make(chan error, len(filesToArchive))

	// Use mutex to protect tar writer (tar format requires sequential writes)
	var mu sync.Mutex

	for _, path := range filesToArchive {
		// Check original context before starting goroutine
		if ctx.Err() != nil {
			return fmt.Errorf("%w: context canceled before archiving", Error)
		}

		path := path // Capture for goroutine

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return fmt.Errorf("%w: context canceled during archiving", Error)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// Get file info
			info, err := os.Lstat(path) // Use Lstat to handle symlinks properly
			if err != nil {
				if os.IsNotExist(err) {
					// File was deleted during archiving, skip it
					return
				}
				errChan <- fmt.Errorf("%w: stat %s: %w", Error, path, err)
				return
			}

			// Create tar header
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				errChan <- fmt.Errorf("%w: create tar header for %s: %w", Error, path, err)
				return
			}

			// Set relative path in archive
			if header.Name, err = filepath.Rel(srcDir, path); err != nil {
				errChan <- fmt.Errorf("%w: get relative path for %s: %w", Error, path, err)
				return
			}

			// Handle symlinks
			if info.Mode()&os.ModeSymlink != 0 {
				if header.Linkname, err = os.Readlink(path); err != nil {
					errChan <- fmt.Errorf("%w: read symlink %s: %w", Error, path, err)
					return
				}
			}

			// Write header and content atomically
			mu.Lock()
			defer mu.Unlock()

			if err := tw.WriteHeader(header); err != nil {
				errChan <- fmt.Errorf("%w: write tar header for %s: %w", Error, path, err)
				return
			}

			// Copy file content for regular files
			if header.Typeflag == tar.TypeReg {
				f, err := os.Open(path)
				if err != nil {
					if os.IsNotExist(err) {
						// File was deleted during archiving
						errChan <- fmt.Errorf("%w: file %s was deleted during archiving", Error, path)
						return
					}
					errChan <- fmt.Errorf("%w: open %s: %w", Error, path, err)
					return
				}
				defer f.Close()

				if _, err := io.Copy(tw, f); err != nil {
					errChan <- fmt.Errorf("%w: copy %s to archive: %w", Error, path, err)
					return
				}
			}
		}()
	}

	// Wait for all archiving to complete
	wg.Wait()
	close(errChan)

	// Check for any archiving errors
	for err := range errChan {
		return fmt.Errorf("%w: archive files: %w", Error, err)
	}

	archiveSuccess = true

	// If not removing source, we're done
	if !removeSrc {
		return nil
	}

	// Remove source files in reverse order (deepest first) to handle directories properly
	sort.Sort(sort.Reverse(sort.StringSlice(filesToArchive)))

	// Use separate context-aware removal with timeout and retries
	return removeSourceFiles(ctx, filesToArchive, srcDir, sem)
}

func removeSourceFiles(ctx context.Context, filesToRemove []string, srcDir string, sem chan struct{}) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(filesToRemove))

	for _, path := range filesToRemove {
		// Check context before each batch
		if ctx.Err() != nil {
			return fmt.Errorf("%w: context canceled before removal", Error)
		}

		path := path // Capture for goroutine

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return fmt.Errorf("%w: context canceled during removal", Error)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			// Retry removal with exponential backoff for transient failures
			maxRetries := 3
			for attempt := 0; attempt < maxRetries; attempt++ {
				if ctx.Err() != nil {
					errChan <- fmt.Errorf("%w: context canceled during removal retry", Error)
					return
				}

				err := os.Remove(path)
				if err == nil || os.IsNotExist(err) {
					return // Success or already removed
				}

				// Check if it's a transient error worth retrying
				if isTransientError(err) && attempt < maxRetries-1 {
					// Exponential backoff: 10ms, 100ms, 1s
					backoff := time.Duration(10*math.Pow(10, float64(attempt))) * time.Millisecond

					select {
					case <-time.After(backoff):
						continue // Retry
					case <-ctx.Done():
						errChan <- fmt.Errorf("%w: context canceled during removal backoff", Error)
						return
					}
				}

				// Final attempt failed or non-transient error
				errChan <- fmt.Errorf("%w: remove %s (attempt %d/%d): %w", Error, path, attempt+1, maxRetries, err)
				return
			}
		}()
	}

	// Wait for all removal operations to complete
	wg.Wait()
	close(errChan)

	// Collect any errors (but don't fail on the first one)
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// Return first error, but log others if you have logging
		return errors[0]
	}

	// Finally, try to remove the source directory itself
	if err := os.Remove(srcDir); err != nil && !os.IsNotExist(err) {
		// If directory is not empty, that's usually fine
		if !isDirectoryNotEmpty(err) {
			return fmt.Errorf("%w: remove source directory %s: %w", Error, srcDir, err)
		}
	}

	return nil
}

// Helper function to check if error indicates directory is not empty
func isDirectoryNotEmpty(err error) bool {
	if pathErr, ok := err.(*os.PathError); ok {
		return pathErr.Err == syscall.ENOTEMPTY || pathErr.Err == syscall.EEXIST
	}
	return false
}

// Helper function to check if an error is transient and worth retrying
func isTransientError(err error) bool {
	if pathErr, ok := err.(*os.PathError); ok {
		switch pathErr.Err {
		case syscall.EBUSY, syscall.ETXTBSY, syscall.EAGAIN:
			return true
		}
	}
	return false
}
