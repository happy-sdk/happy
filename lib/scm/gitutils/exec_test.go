// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gitutils

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/sdk/session"
)

// withFakeExecRaw substitutes execRaw for the duration of the test with fn,
// which decides the (output, error) for a command purely from its Args -
// this exercises this package's own argument-building and output-parsing
// logic without a real session or a real git binary.
func withFakeExecRaw(t *testing.T, fn func(cmd *exec.Cmd) ([]byte, error)) {
	t.Helper()
	orig := execRaw
	execRaw = func(_ *session.Context, cmd *exec.Cmd) ([]byte, error) {
		return fn(cmd)
	}
	t.Cleanup(func() { execRaw = orig })
}

// withFakeExecRun substitutes execRun the same way, for the Run-based
// functions (Commit, Tag).
func withFakeExecRun(t *testing.T, fn func(cmd *exec.Cmd) error) {
	t.Helper()
	orig := execRun
	execRun = func(_ *session.Context, cmd *exec.Cmd) error {
		return fn(cmd)
	}
	t.Cleanup(func() { execRun = orig })
}

func TestDirty(t *testing.T) {
	t.Run("dirty when status has output", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte(" M main.go\n"), nil
		})
		if !Dirty(nil, "/repo", ".") {
			t.Error("expected non-empty status output to be reported as dirty")
		}
	})

	t.Run("clean when status is empty", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("  \n"), nil
		})
		if Dirty(nil, "/repo", ".") {
			t.Error("expected blank status output to be reported as clean")
		}
	})

	t.Run("false on exec error", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return nil, errors.New("boom")
		})
		if Dirty(nil, "/repo", ".") {
			t.Error("expected an exec error to be reported as clean")
		}
	})
}

func TestCurrentBranch(t *testing.T) {
	t.Run("trims and returns the branch", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte(" main \n"), nil
		})
		got, err := CurrentBranch(nil, "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "main" {
			t.Errorf("expected %q, got %q", "main", got)
		}
	})

	t.Run("wraps exec errors with Error", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return nil, errors.New("not a git repository")
		})
		_, err := CurrentBranch(nil, "/repo")
		if !errors.Is(err, Error) {
			t.Errorf("expected error to wrap gitutils.Error, got: %v", err)
		}
	})
}

func TestCurrentRemote(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			switch {
			case containsArg(cmd, "@{u}"):
				return []byte("origin/main\n"), nil
			case containsArg(cmd, "--get"):
				return []byte("git@github.com:happy-sdk/happy.git\n"), nil
			}
			t.Fatalf("unexpected command: %v", cmd.Args)
			return nil, nil
		})

		name, url, err := CurrentRemote(nil, "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "origin" {
			t.Errorf("expected remote name %q, got %q", "origin", name)
		}
		if url != "git@github.com:happy-sdk/happy.git" {
			t.Errorf("unexpected remote url: %q", url)
		}
	})

	t.Run("propagates upstream lookup error without querying url", func(t *testing.T) {
		calls := 0
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			calls++
			return nil, errors.New("no upstream configured")
		})
		_, _, err := CurrentRemote(nil, "/repo")
		if !errors.Is(err, Error) {
			t.Errorf("expected error to wrap gitutils.Error, got: %v", err)
		}
		if calls != 1 {
			t.Errorf("expected only the upstream lookup to run, got %d exec calls", calls)
		}
	})

	t.Run("propagates url lookup error", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			if containsArg(cmd, "@{u}") {
				return []byte("origin/main\n"), nil
			}
			return nil, errors.New("no such config key")
		})
		_, _, err := CurrentRemote(nil, "/repo")
		if !errors.Is(err, Error) {
			t.Errorf("expected error to wrap gitutils.Error, got: %v", err)
		}
	})
}

func TestTagRefMatchesLocal(t *testing.T) {
	tests := []struct {
		name   string
		output string
		tag    string
		want   bool
	}{
		{"exact match", "v1.2.0\n", "v1.2.0", true},
		{"no match", "", "v1.2.0", false},
		{"prefix must not match", "v1.2.0\n", "v1.2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tagRefMatches(tt.output, tt.tag); got != tt.want {
				t.Errorf("tagRefMatches(%q, %q) = %v, want %v", tt.output, tt.tag, got, tt.want)
			}
		})
	}
}

func TestTagRefMatchesRemote(t *testing.T) {
	tests := []struct {
		name   string
		output string
		tag    string
		want   bool
	}{
		{"exact ref match", "abc123\trefs/tags/v1.2.0\n", "v1.2.0", true},
		{"peeled annotated tag match", "abc123\trefs/tags/v1.2.0^{}\n", "v1.2.0", true},
		{
			"prefix must not match a longer tag",
			"abc123\trefs/tags/v1.2.0\n",
			"v1.2",
			false,
		},
		{"empty output", "", "v1.2.0", false},
		{"blank lines are skipped", "\n\nabc123\trefs/tags/v1.2.0\n", "v1.2.0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tagRefMatches(tt.output, tt.tag); got != tt.want {
				t.Errorf("tagRefMatches(%q, %q) = %v, want %v", tt.output, tt.tag, got, tt.want)
			}
		})
	}
}

func TestRemoteTagExists(t *testing.T) {
	t.Run("true on match", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("abc123\trefs/tags/v1.2.0\n"), nil
		})
		if !RemoteTagExists(nil, "/repo", "origin", "v1.2.0") {
			t.Error("expected tag to be reported as existing")
		}
	})

	t.Run("false on exec error", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return nil, errors.New("network error")
		})
		if RemoteTagExists(nil, "/repo", "origin", "v1.2.0") {
			t.Error("expected an exec error to be reported as tag not existing")
		}
	})
}

func TestTagExists(t *testing.T) {
	t.Run("true on match", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte("v1.2.0\n"), nil
		})
		if !TagExists(nil, "/repo", "v1.2.0") {
			t.Error("expected tag to be reported as existing")
		}
	})

	t.Run("false when not found", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte(""), nil
		})
		if TagExists(nil, "/repo", "v1.2.0") {
			t.Error("expected an empty listing to be reported as tag not existing")
		}
	})

	t.Run("false on exec error", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return nil, errors.New("not a git repository")
		})
		if TagExists(nil, "/repo", "v1.2.0") {
			t.Error("expected an exec error to be reported as tag not existing")
		}
	})
}

func TestCommit(t *testing.T) {
	t.Run("no-op when clean", func(t *testing.T) {
		calls := 0
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			calls++
			return []byte(""), nil // clean status
		})
		withFakeExecRun(t, func(cmd *exec.Cmd) error {
			t.Fatal("expected no commands to run when the tree is clean")
			return nil
		})
		if err := Commit(nil, "/repo", []string{"-A"}, "msg"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Errorf("expected exactly one dirty-check call, got %d", calls)
		}
	})

	t.Run("adds then commits when dirty", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte(" M main.go\n"), nil
		})
		var ran []string
		withFakeExecRun(t, func(cmd *exec.Cmd) error {
			ran = append(ran, strings.Join(cmd.Args, " "))
			return nil
		})
		if err := Commit(nil, "/repo", []string{"-A"}, "my message"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ran) != 2 {
			t.Fatalf("expected 2 commands to run, got %d: %v", len(ran), ran)
		}
		if !strings.Contains(ran[0], "add -A") {
			t.Errorf("expected first command to be `git add -A`, got %q", ran[0])
		}
		if !strings.Contains(ran[1], "commit -sm my message") {
			t.Errorf("expected second command to be the commit, got %q", ran[1])
		}
	})

	t.Run("propagates add failure without committing", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte(" M main.go\n"), nil
		})
		calls := 0
		withFakeExecRun(t, func(cmd *exec.Cmd) error {
			calls++
			return errors.New("add failed")
		})
		err := Commit(nil, "/repo", []string{"-A"}, "msg")
		if !errors.Is(err, Error) {
			t.Errorf("expected error to wrap gitutils.Error, got: %v", err)
		}
		if calls != 1 {
			t.Errorf("expected commit to not run after add fails, got %d run calls", calls)
		}
	})

	t.Run("propagates commit failure", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			return []byte(" M main.go\n"), nil
		})
		calls := 0
		withFakeExecRun(t, func(cmd *exec.Cmd) error {
			calls++
			if calls == 1 {
				return nil // add succeeds
			}
			return errors.New("commit failed")
		})
		err := Commit(nil, "/repo", []string{"-A"}, "msg")
		if !errors.Is(err, Error) {
			t.Errorf("expected error to wrap gitutils.Error, got: %v", err)
		}
	})
}

func TestTag(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var ran string
		withFakeExecRun(t, func(cmd *exec.Cmd) error {
			ran = strings.Join(cmd.Args, " ")
			return nil
		})
		if err := Tag(nil, "/repo", "v1.2.0", "release message"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(ran, "tag -s v1.2.0 -m release message") {
			t.Errorf("unexpected command: %q", ran)
		}
	})

	t.Run("propagates failure", func(t *testing.T) {
		withFakeExecRun(t, func(cmd *exec.Cmd) error {
			return errors.New("gpg signing failed")
		})
		err := Tag(nil, "/repo", "v1.2.0", "release message")
		if !errors.Is(err, Error) {
			t.Errorf("expected error to wrap gitutils.Error, got: %v", err)
		}
	})
}

func containsArg(cmd *exec.Cmd, arg string) bool {
	return slices.Contains(cmd.Args, arg)
}
