// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/sdk/session"
)

func newTestSession(t *testing.T) (*session.Context, func()) {
	t.Helper()
	sess, _, cleanup, err := session.CreateTestSession(nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	return sess, cleanup
}

func TestExec(t *testing.T) {
	sess, cleanup := newTestSession(t)
	defer cleanup()

	cmd := exec.Command("echo", "hello")
	out, err := Exec(sess, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("Exec output = %q, want %q", out, "hello")
	}
}

func TestExecRaw(t *testing.T) {
	sess, cleanup := newTestSession(t)
	defer cleanup()

	cmd := exec.Command("echo", "hello")
	out, err := ExecRaw(sess, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(string(out)) != "hello" {
		t.Errorf("ExecRaw output = %q, want %q", string(out), "hello")
	}
}

func TestExecRawCommandError(t *testing.T) {
	sess, cleanup := newTestSession(t)
	defer cleanup()

	cmd := exec.Command("sh", "-c", "echo failmsg >&2; exit 1")
	_, err := ExecRaw(sess, cmd)
	if err == nil {
		t.Fatal("expected an error for a failing command")
	}
	if !strings.Contains(err.Error(), "failmsg") {
		t.Errorf("expected error to include stderr output, got: %v", err)
	}
}

func TestRun(t *testing.T) {
	sess, cleanup := newTestSession(t)
	defer cleanup()

	cmd := exec.Command("echo", "hello-run")
	if err := Run(sess, cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCommandError(t *testing.T) {
	sess, cleanup := newTestSession(t)
	defer cleanup()

	cmd := exec.Command("sh", "-c", "exit 1")
	if err := Run(sess, cmd); err == nil {
		t.Fatal("expected an error for a failing command")
	}
}

// withStdin temporarily redirects os.Stdin to a pipe pre-filled with input,
// restoring the original os.Stdin when the returned cleanup function runs.
func withStdin(t *testing.T, input string) func() {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("failed to write to pipe: %v", err)
	}
	_ = w.Close()

	orig := os.Stdin
	os.Stdin = r
	return func() {
		os.Stdin = orig
		_ = r.Close()
	}
}

func TestAskForConfirmation(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			restore := withStdin(t, tt.input)
			defer restore()

			got := AskForConfirmation("question?")
			if got != tt.want {
				t.Errorf("AskForConfirmation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAskForInput(t *testing.T) {
	restore := withStdin(t, "my answer\n")
	defer restore()

	got := AskForInput("question?")
	if got != "my answer" {
		t.Errorf("AskForInput() = %q, want %q", got, "my answer")
	}
}
