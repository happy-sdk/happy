// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package exec

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestInterpolate(t *testing.T) {
	inputs := map[string]any{
		"module":    "pkg/vars",
		"threshold": float64(95),
		"ratio":     1.5,
		"on":        true,
		"empty":     "",
	}

	for _, tt := range []struct {
		name, in, want string
	}{
		{"plain", "go test", "go test"},
		{"simple", "{{ .module }}", "pkg/vars"},
		{"no spaces", "{{.module}}", "pkg/vars"},
		{"embedded", "./{{ .module }}/...", "./pkg/vars/..."},
		{"integer renders without decimals", "{{ .threshold }}", "95"},
		{"non-integer keeps precision", "{{ .ratio }}", "1.5"},
		{"bool", "{{ .on }}", "true"},
		{"empty stays empty", "{{ .empty }}", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interpolate(tt.in, inputs, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// A placeholder naming a field the schema does not declare is a manifest bug.
// Substituting an empty string would produce a command with a hole in it and
// run it anyway, which is the worst possible way to surface that.
func TestInterpolateRejectsUndeclaredField(t *testing.T) {
	_, err := interpolate("{{ .nope }}", map[string]any{"module": "pkg/vars"}, nil)
	if !errors.Is(err, ErrUnknownArg) {
		t.Fatalf("got %v, want ErrUnknownArg", err)
	}
}

// Inputs are substituted into argv entries, never parsed as a command line, so
// shell metacharacters are inert data.
func TestInterpolateDoesNotInterpretShellMetacharacters(t *testing.T) {
	got, err := interpolate("{{ .module }}", map[string]any{"module": "pkg/vars; rm -rf /"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pkg/vars; rm -rf /" {
		t.Fatalf("got %q, want the value passed through verbatim", got)
	}
}

func TestResolveCwdConfinesToRepository(t *testing.T) {
	root := t.TempDir()

	t.Run("relative stays inside", func(t *testing.T) {
		got, err := resolveCwd(root, "pkg/vars")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := filepath.Join(root, "pkg", "vars"); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("empty means repository root", func(t *testing.T) {
		got, err := resolveCwd(root, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != root {
			t.Fatalf("got %q, want %q", got, root)
		}
	})

	for _, cwd := range []string{"..", "../sibling", "pkg/../../escape", "/etc"} {
		t.Run("rejects "+cwd, func(t *testing.T) {
			if _, err := resolveCwd(root, cwd); !errors.Is(err, ErrOutsideRepo) {
				t.Fatalf("cwd %q: got %v, want ErrOutsideRepo", cwd, err)
			}
		})
	}
}

func TestRunCapturesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell-free echo equivalent")
	}
	res, err := Run(t.Context(), Request{
		RepoDir: t.TempDir(),
		Command: []string{"echo", "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(res.Output) != "hello" {
		t.Fatalf("got %q, want %q", res.Output, "hello")
	}
	if res.ExitCode != 0 {
		t.Fatalf("got exit %d, want 0", res.ExitCode)
	}
}

// A non-zero exit is the answer to "do the tests pass", not a failure of the
// tool, so it must come back as a result with output intact.
func TestRunReportsExitCodeWithoutError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only helper")
	}
	res, err := Run(t.Context(), Request{
		RepoDir: t.TempDir(),
		Command: []string{"sh", "-c", "echo failing >&2; exit 3"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 3 {
		t.Fatalf("got exit %d, want 3", res.ExitCode)
	}
	if !strings.Contains(res.Output, "failing") {
		t.Fatalf("stderr should be captured, got %q", res.Output)
	}
}

func TestRunAppliesConditionalArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only helper")
	}
	req := Request{
		RepoDir:  t.TempDir(),
		Command:  []string{"echo", "base"},
		Args:     []ArgRule{{If: "{{ .run }}", Add: []string{"-run", "{{ .run }}"}}},
		Declared: map[string]bool{"run": true},
	}

	req.Inputs = map[string]any{"run": ""}
	res, err := Run(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(res.Output) != "base" {
		t.Fatalf("empty condition must add nothing, got %q", res.Output)
	}

	req.Inputs = map[string]any{"run": "TestX"}
	res, err = Run(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(res.Output) != "base -run TestX" {
		t.Fatalf("got %q, want the conditional args appended", res.Output)
	}
}

func TestRunTimesOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-only helper")
	}
	_, err := Run(t.Context(), Request{
		RepoDir: t.TempDir(),
		Command: []string{"sleep", "30"},
		Timeout: 150 * time.Millisecond,
	})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("got %v, want ErrTimeout", err)
	}
}

func TestRunRejectsEscapingCwd(t *testing.T) {
	_, err := Run(context.Background(), Request{
		RepoDir: t.TempDir(),
		Command: []string{"echo", "hi"},
		Cwd:     "../..",
	})
	if !errors.Is(err, ErrOutsideRepo) {
		t.Fatalf("got %v, want ErrOutsideRepo", err)
	}
}
