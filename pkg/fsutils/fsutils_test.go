// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package fsutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func TestAvailableSpace(t *testing.T) {
	// Test with current directory
	space, err := AvailableSpace(".")
	testutils.NoError(t, err, "expected no error getting available space")
	testutils.Assert(t, space > 0, "expected available space to be greater than 0")

	// Test with non-existent path
	_, err = AvailableSpace("/nonexistent/path/that/does/not/exist")
	testutils.Error(t, err, "expected error for non-existent path")
}

func TestCountFilesAndDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure
	testDir := filepath.Join(tmpDir, "test")
	err := os.MkdirAll(testDir, 0755)
	testutils.NoError(t, err)

	subDir := filepath.Join(testDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	testutils.NoError(t, err)

	// Create files
	file1 := filepath.Join(testDir, "file1.txt")
	err = os.WriteFile(file1, []byte("content"), 0644)
	testutils.NoError(t, err)

	file2 := filepath.Join(subDir, "file2.txt")
	err = os.WriteFile(file2, []byte("content"), 0644)
	testutils.NoError(t, err)

	// Count files and directories
	filec, dirc, err := CountFilesAndDirs(testDir)
	testutils.NoError(t, err, "expected no error counting files and dirs")
	testutils.Equal(t, 2, filec, "expected 2 files")
	testutils.Equal(t, 2, dirc, "expected 2 directories")

	// Test with non-existent directory
	_, _, err = CountFilesAndDirs("/nonexistent/directory")
	testutils.Error(t, err, "expected error for non-existent directory")
}

func TestFileStat(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	err := os.WriteFile(tmpFile, []byte("test content"), 0644)
	testutils.NoError(t, err)

	file, err := os.Open(tmpFile)
	testutils.NoError(t, err)
	defer func() { _ = file.Close() }()

	info, err := FileStat(file)
	testutils.NoError(t, err, "expected no error getting file stat")
	testutils.Equal(t, "testfile.txt", info.Name)
	testutils.Assert(t, info.Size > 0, "expected file size to be greater than 0")
	testutils.Assert(t, !info.Mtime.IsZero(), "expected modification time to be set")
	testutils.Assert(t, !info.Atime.IsZero(), "expected access time to be set")
	testutils.Assert(t, !info.Ctime.IsZero(), "expected change time to be set")
	testutils.Assert(t, !info.Btime.IsZero(), "expected birth time to be set")
}

func TestStat(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	err := os.WriteFile(tmpFile, []byte("test content"), 0644)
	testutils.NoError(t, err)

	info, err := Stat(tmpFile)
	testutils.NoError(t, err, "expected no error getting stat")
	testutils.Equal(t, "testfile.txt", info.Name)
	testutils.Assert(t, info.Size > 0, "expected file size to be greater than 0")
	testutils.Assert(t, !info.Mtime.IsZero(), "expected modification time to be set")

	// Test with non-existent file
	_, err = Stat("/nonexistent/file")
	testutils.Error(t, err, "expected error for non-existent file")
}

func TestIsRegular(t *testing.T) {
	tmpDir := t.TempDir()

	// Test regular file
	file := filepath.Join(tmpDir, "regular.txt")
	err := os.WriteFile(file, []byte("content"), 0644)
	testutils.NoError(t, err)
	testutils.Assert(t, IsRegular(file), "expected regular file to return true")

	// Test directory
	testutils.Assert(t, !IsRegular(tmpDir), "expected directory to return false")

	// Test non-existent path
	testutils.Assert(t, !IsRegular("/nonexistent/path"), "expected non-existent path to return false")
}

func TestIsSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular file
	file := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(file, []byte("content"), 0644)
	testutils.NoError(t, err)
	testutils.Assert(t, !IsSymlink(file), "expected regular file to return false")

	// Create a symlink (if supported on platform)
	link := filepath.Join(tmpDir, "link.txt")
	err = os.Symlink(file, link)
	if err == nil {
		testutils.Assert(t, IsSymlink(link), "expected symlink to return true")
	}

	// Test non-existent path
	testutils.Assert(t, !IsSymlink("/nonexistent/path"), "expected non-existent path to return false")
}

func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Test directory
	testutils.Assert(t, IsDir(tmpDir), "expected directory to return true")

	// Test regular file
	file := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(file, []byte("content"), 0644)
	testutils.NoError(t, err)
	testutils.Assert(t, !IsDir(file), "expected regular file to return false")

	// Test non-existent path
	testutils.Assert(t, !IsDir("/nonexistent/path"), "expected non-existent path to return false")
}

func TestSELinuxContext(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	err := os.WriteFile(tmpFile, []byte("content"), 0644)
	testutils.NoError(t, err)

	context, err := SELinuxContext(tmpFile)
	if err == nil {
		_ = context
	}
}

func TestFileSELinuxContext(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile.txt")
	err := os.WriteFile(tmpFile, []byte("content"), 0644)
	testutils.NoError(t, err)

	file, err := os.Open(tmpFile)
	testutils.NoError(t, err)
	defer func() { _ = file.Close() }()

	context, err := FileSELinuxContext(file)
	if err == nil {
		_ = context
	}
}

func TestUserDataDir(t *testing.T) {
	appslug := "test-app"
	dir := UserDataDir(appslug)
	testutils.Assert(t, dir != "", "expected non-empty directory path")
	testutils.Assert(t, filepath.IsAbs(dir), "expected absolute path")
}

func TestUserRuntimeDir(t *testing.T) {
	appslug := "test-app"
	dir := UserRuntimeDir(appslug)
	testutils.Assert(t, dir != "", "expected non-empty directory path")
	testutils.Assert(t, filepath.IsAbs(dir), "expected absolute path")
}

func TestUserStateDir(t *testing.T) {
	appslug := "test-app"
	dir := UserStateDir(appslug)
	testutils.Assert(t, dir != "", "expected non-empty directory path")
	testutils.Assert(t, filepath.IsAbs(dir), "expected absolute path")
}

func TestDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with known sizes
	file1 := filepath.Join(tmpDir, "file1.txt")
	err := os.WriteFile(file1, []byte("12345"), 0644)
	testutils.NoError(t, err)

	subDir := filepath.Join(tmpDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	testutils.NoError(t, err)

	file2 := filepath.Join(subDir, "file2.txt")
	err = os.WriteFile(file2, []byte("67890"), 0644)
	testutils.NoError(t, err)

	size, err := DirSize(tmpDir)
	testutils.NoError(t, err, "expected no error getting directory size")
	testutils.Assert(t, size >= 10, "expected size to be at least 10 bytes")

	// Test with non-existent directory
	_, err = DirSize("/nonexistent/directory")
	testutils.Error(t, err, "expected error for non-existent directory")
}

func TestDirBtimeSpan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with some delay
	file1 := filepath.Join(tmpDir, "file1.txt")
	err := os.WriteFile(file1, []byte("content1"), 0644)
	testutils.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	file2 := filepath.Join(tmpDir, "file2.txt")
	err = os.WriteFile(file2, []byte("content2"), 0644)
	testutils.NoError(t, err)

	// Test non-recursive
	oldest, newest, span, err := DirBtimeSpan(tmpDir, false)
	testutils.NoError(t, err, "expected no error getting btime span")
	testutils.Assert(t, !oldest.IsZero(), "expected oldest time to be set")
	testutils.Assert(t, !newest.IsZero(), "expected newest time to be set")
	testutils.Assert(t, span >= 0, "expected span to be non-negative")

	// Test recursive
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	testutils.NoError(t, err)

	file3 := filepath.Join(subDir, "file3.txt")
	err = os.WriteFile(file3, []byte("content3"), 0644)
	testutils.NoError(t, err)

	oldest, newest, span, err = DirBtimeSpan(tmpDir, true)
	testutils.NoError(t, err, "expected no error getting recursive btime span")
	testutils.Assert(t, !oldest.IsZero(), "expected oldest time to be set")
	testutils.Assert(t, !newest.IsZero(), "expected newest time to be set")
	testutils.Assert(t, span >= 0, "expected span to be non-negative")

	// Test with empty directory
	emptyDir := t.TempDir()
	_, _, _, err = DirBtimeSpan(emptyDir, false)
	testutils.Error(t, err, "expected error for empty directory")
}

func TestIsStdoutStderrFile(t *testing.T) {
	result, err := IsStdoutStderrFile("/dev/stdout")
	_ = result
	_ = err
}
