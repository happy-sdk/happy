// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package goutils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainsGoModfile(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		dir := t.TempDir()
		gomod := filepath.Join(dir, "go.mod")
		if err := os.WriteFile(gomod, []byte("module example.com/test\n\ngo 1.25\n"), 0644); err != nil {
			t.Fatal(err)
		}

		got, ok := ContainsGoModfile(dir)
		if !ok {
			t.Fatal("expected go.mod to be found")
		}
		if got != gomod {
			t.Errorf("got %q, want %q", got, gomod)
		}
	})

	t.Run("absent", func(t *testing.T) {
		dir := t.TempDir()
		_, ok := ContainsGoModfile(dir)
		if ok {
			t.Error("expected no go.mod to be found")
		}
	})
}

func writeGoMod(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDependsOnHappy(t *testing.T) {
	t.Run("no go.mod", func(t *testing.T) {
		dir := t.TempDir()
		_, yes, err := DependsOnHappy(dir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if yes {
			t.Error("expected yes=false when there is no go.mod")
		}
	})

	t.Run("is happy itself", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "module github.com/happy-sdk/happy\n\ngo 1.25\n")
		_, yes, err := DependsOnHappy(dir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !yes {
			t.Error("expected yes=true for the happy module itself")
		}
	})

	t.Run("requires happy", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "module example.com/test\n\ngo 1.25\n\nrequire github.com/happy-sdk/happy v1.2.3\n")
		ver, yes, err := DependsOnHappy(dir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !yes {
			t.Fatal("expected yes=true when go.mod requires happy")
		}
		if ver.String() != "v1.2.3" {
			t.Errorf("ver = %q, want %q", ver.String(), "v1.2.3")
		}
	})

	t.Run("does not require happy", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "module example.com/test\n\ngo 1.25\n\nrequire example.com/other v1.0.0\n")
		_, yes, err := DependsOnHappy(dir)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if yes {
			t.Error("expected yes=false when go.mod does not require happy")
		}
	})

	t.Run("malformed go.mod", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "this is not a valid go.mod file {{{")
		_, _, err := DependsOnHappy(dir)
		if err == nil {
			t.Error("expected an error for a malformed go.mod")
		}
	})
}

// TestIsGoRun confirms IsGoRun returns true when invoked from a `go test`
// binary, which (like `go run`) is compiled into a temp build directory.
func TestIsGoRun(t *testing.T) {
	if !IsGoRun() {
		t.Error("expected IsGoRun() to report true under `go test`")
	}
}
