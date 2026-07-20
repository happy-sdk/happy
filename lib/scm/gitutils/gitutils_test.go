// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gitutils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsRepository(t *testing.T) {
	t.Run("true when .git exists", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0750); err != nil {
			t.Fatal(err)
		}
		if !IsRepository(dir) {
			t.Error("expected a directory with a .git subdirectory to be a repository")
		}
	})

	t.Run("false when .git is absent", func(t *testing.T) {
		dir := t.TempDir()
		if IsRepository(dir) {
			t.Error("expected a directory with no .git to not be a repository")
		}
	})

	t.Run("false on stat error other than not-exist", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permission bits behave differently on windows")
		}
		if os.Geteuid() == 0 {
			t.Skip("root ignores permission bits, can't provoke a permission error")
		}
		parent := t.TempDir()
		if err := os.Chmod(parent, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(parent, 0750) })

		// os.Stat(parent+"/.git") fails with a permission error here, not
		// ErrNotExist - IsRepository must not treat that as "is a
		// repository" just because the error isn't ErrNotExist.
		if IsRepository(parent) {
			t.Error("expected a permission error to not be reported as a repository")
		}
	})
}

func TestFindRepositoryRoot(t *testing.T) {
	t.Run("finds an ancestor repository root", func(t *testing.T) {
		root := t.TempDir()
		if err := os.Mkdir(filepath.Join(root, ".git"), 0750); err != nil {
			t.Fatal(err)
		}
		nested := filepath.Join(root, "a", "b", "c")
		if err := os.MkdirAll(nested, 0750); err != nil {
			t.Fatal(err)
		}

		dir, found, err := FindRepositoryRoot(nested)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found {
			t.Fatal("expected to find the repository root")
		}
		wantRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		gotRoot, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatal(err)
		}
		if gotRoot != wantRoot {
			t.Errorf("expected root %s, got %s", wantRoot, gotRoot)
		}
	})

	t.Run("returns not found when no ancestor is a repository", func(t *testing.T) {
		dir := t.TempDir()
		_, found, err := FindRepositoryRoot(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found {
			t.Error("expected not to find a repository root")
		}
	})
}

func TestNewIgnoreMatcher(t *testing.T) {
	m := NewIgnoreMatcher([]string{"*.log", "build/"}, nil)

	if !m.Match([]string{"debug.log"}, false) {
		t.Error("expected debug.log to be matched by *.log")
	}
	if !m.Match([]string{"build"}, true) {
		t.Error("expected build/ dir to be matched")
	}
	if m.Match([]string{"main.go"}, false) {
		t.Error("expected main.go to not be matched")
	}
}

func TestNewConfig(t *testing.T) {
	spec, err := NewConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, key := range []string{
		"repo.found",
		"loaded",
		"repo.root",
		"repo.branch",
		"repo.remote.name",
		"repo.remote.url",
		"repo.dirty",
		"committer.name",
		"committer.email",
	} {
		if !spec.Accepts(key) {
			t.Errorf("expected config to accept key %q", key)
		}
	}

	t.Run("repo.root accepts empty value", func(t *testing.T) {
		if err := spec.Set("repo.root", ""); err != nil {
			t.Errorf("expected empty repo.root to be accepted, got: %v", err)
		}
	})

	t.Run("repo.root rejects a non-repository path", func(t *testing.T) {
		if err := spec.Set("repo.root", t.TempDir()); err == nil {
			t.Error("expected a non-repository path to be rejected")
		}
	})

	t.Run("repo.root accepts a real repository path", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0750); err != nil {
			t.Fatal(err)
		}
		if err := spec.Set("repo.root", dir); err != nil {
			t.Errorf("expected a valid repository path to be accepted, got: %v", err)
		}
	})
}
