// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gitutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	git "github.com/go-git/go-git/v6"
)

// initRepo creates a real, structurally valid (non-bare) git repository at
// dir via go-git itself - no subprocess, no reliance on a git binary being
// on PATH - so tests can distinguish "a real repository" from "a directory
// that merely has something named .git in it".
func initRepo(t *testing.T, dir string) {
	t.Helper()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("failed to init fixture repository: %v", err)
	}
}

func TestIsRepository(t *testing.T) {
	t.Run("true for a real repository", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir)
		if !IsRepository(dir) {
			t.Error("expected a real git repository to be recognized as one")
		}
	})

	t.Run("false for a dummy .git directory", func(t *testing.T) {
		// A bare empty ".git" directory is not a valid repository - it's
		// missing HEAD, objects, refs, etc. This is the false positive
		// IsRepository must not report.
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0750); err != nil {
			t.Fatal(err)
		}
		if IsRepository(dir) {
			t.Error("expected an empty dummy .git directory to not be recognized as a repository")
		}
	})

	t.Run("false when .git is absent", func(t *testing.T) {
		dir := t.TempDir()
		if IsRepository(dir) {
			t.Error("expected a directory with no .git to not be a repository")
		}
	})

	t.Run("false for a nonexistent path", func(t *testing.T) {
		if IsRepository(filepath.Join(t.TempDir(), "does-not-exist")) {
			t.Error("expected a nonexistent path to not be a repository")
		}
	})
}

func TestFindRepositoryRoot(t *testing.T) {
	t.Run("finds an ancestor repository root", func(t *testing.T) {
		root := t.TempDir()
		initRepo(t, root)
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

func TestClone(t *testing.T) {
	t.Run("runs git clone into the destination", func(t *testing.T) {
		var got []string
		withFakeExecRun(t, func(cmd *exec.Cmd) error {
			got = cmd.Args
			return nil
		})

		dir := filepath.Join(t.TempDir(), "nested", "repo")
		if err := Clone(nil, "git@example.com:acme/repo.git", dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"git", "clone", "git@example.com:acme/repo.git", dir}
		if !slices.Equal(got, want) {
			t.Fatalf("args = %v, want %v", got, want)
		}
		// The parent is created so callers need not prepare the destination.
		if fi, err := os.Stat(filepath.Dir(dir)); err != nil || !fi.IsDir() {
			t.Fatalf("parent directory not created: %v", err)
		}
	})

	t.Run("rejects an empty remote", func(t *testing.T) {
		if err := Clone(nil, "", t.TempDir()); err == nil {
			t.Fatal("expected an error for an empty remote")
		}
	})
}
