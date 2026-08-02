// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

// gitInit initializes a repository at dir, skipping the test when no git
// binary is available. go-git is only an indirect dependency of this module,
// so shelling out keeps it that way.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(git, "init", "-q", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v: %s", dir, err, out)
	}
}

func writeConfig(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte("version: \"1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A repository checked out inside another repository is its own project. This
// is the happy-sdk workspace layout: an outer repository holding each project
// under a gitignored src/ directory. Without a repository boundary the inner
// project resolves to the outer root and its .happy.yaml is never read.
func TestFindProjectDirStopsAtNestedRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)

	inner := filepath.Join(root, "src", "proj")
	gitInit(t, inner)
	writeConfig(t, inner)

	dir, found, err := FindProjectDir(inner)
	testutils.NoError(t, err)
	testutils.Assert(t, found, "expected a project to be found")
	testutils.Equal(t, inner, dir, "nested repository must resolve to itself, not the outer repository")
}

// Ascending from a subdirectory of a nested repository must still stop at that
// repository, not escape into the outer one.
func TestFindProjectDirAscendsWithinNestedRepository(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)

	inner := filepath.Join(root, "src", "proj")
	gitInit(t, inner)
	writeConfig(t, inner)

	sub := filepath.Join(inner, "pkg", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	dir, found, err := FindProjectDir(sub)
	testutils.NoError(t, err)
	testutils.Assert(t, found, "expected a project to be found")
	testutils.Equal(t, inner, dir, "expected the nested repository root")
}

// The monorepo case the outermost-match behaviour exists for: a module
// directory inside a repository is not a repository root, so it must keep
// resolving up to the repository root.
func TestFindProjectDirResolvesMonorepoModuleToRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	writeConfig(t, root)

	module := filepath.Join(root, "pkg", "vars")
	if err := os.MkdirAll(module, 0o755); err != nil {
		t.Fatal(err)
	}
	// A nested module carrying its own config must not shadow the repository
	// root - only a repository boundary stops the ascent.
	writeConfig(t, module)

	dir, found, err := FindProjectDir(module)
	testutils.NoError(t, err)
	testutils.Assert(t, found, "expected a project to be found")
	testutils.Equal(t, root, dir, "a module directory must resolve to the repository root")
}

func TestFindProjectDirNotFound(t *testing.T) {
	dir := t.TempDir()

	got, found, err := FindProjectDir(dir)
	testutils.NoError(t, err)
	testutils.Assert(t, !found, "expected no project in an empty directory")
	testutils.Equal(t, dir, got, "expected the original working directory back")
}
