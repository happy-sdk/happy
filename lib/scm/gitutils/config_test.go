// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gitutils

import (
	"errors"
	"os/exec"
	"testing"
)

func TestConfigBlueprint(t *testing.T) {
	c := &Config{}
	bp, err := c.Blueprint()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bp == nil {
		t.Fatal("expected a non-nil blueprint")
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			switch {
			case containsArg(cmd, "user.name"):
				return []byte("Jane Doe\n"), nil
			case containsArg(cmd, "user.email"):
				return []byte("jane@example.com\n"), nil
			case containsArg(cmd, "HEAD"):
				return []byte("main\n"), nil
			case containsArg(cmd, "@{u}"):
				return []byte("origin/main\n"), nil
			case containsArg(cmd, "--get"):
				return []byte("git@github.com:happy-sdk/happy.git\n"), nil
			}
			t.Fatalf("unexpected command: %v", cmd.Args)
			return nil, nil
		})

		cnf, err := LoadConfig(nil, "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cnf.CommitterName != "Jane Doe" || cnf.AuthorName != "Jane Doe" {
			t.Errorf("unexpected committer/author name: %q/%q", cnf.CommitterName, cnf.AuthorName)
		}
		if cnf.CommitterEmail != "jane@example.com" || cnf.AuthorEmail != "jane@example.com" {
			t.Errorf("unexpected committer/author email: %q/%q", cnf.CommitterEmail, cnf.AuthorEmail)
		}
		if cnf.Branch != "main" {
			t.Errorf("unexpected branch: %q", cnf.Branch)
		}
		if cnf.RemoteName != "origin" {
			t.Errorf("unexpected remote name: %q", cnf.RemoteName)
		}
		if cnf.RemoteURL != "git@github.com:happy-sdk/happy.git" {
			t.Errorf("unexpected remote url: %q", cnf.RemoteURL)
		}
	})

	t.Run("degrades independently when identity is unavailable", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			switch {
			case containsArg(cmd, "user.name"), containsArg(cmd, "user.email"):
				return nil, errors.New("no such config key")
			case containsArg(cmd, "HEAD"):
				return []byte("main\n"), nil
			case containsArg(cmd, "@{u}"):
				return []byte("origin/main\n"), nil
			case containsArg(cmd, "--get"):
				return []byte("git@github.com:happy-sdk/happy.git\n"), nil
			}
			t.Fatalf("unexpected command: %v", cmd.Args)
			return nil, nil
		})

		cnf, err := LoadConfig(nil, "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cnf.CommitterName != "" || cnf.CommitterEmail != "" {
			t.Errorf("expected empty committer identity, got %q/%q", cnf.CommitterName, cnf.CommitterEmail)
		}
		if cnf.Branch != "main" {
			t.Errorf("expected branch to still be discovered, got %q", cnf.Branch)
		}
	})

	t.Run("degrades independently when there is no remote", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			switch {
			case containsArg(cmd, "user.name"):
				return []byte("Jane Doe\n"), nil
			case containsArg(cmd, "user.email"):
				return []byte("jane@example.com\n"), nil
			case containsArg(cmd, "HEAD"):
				return []byte("main\n"), nil
			case containsArg(cmd, "@{u}"):
				return nil, errors.New("no upstream configured")
			}
			t.Fatalf("unexpected command: %v", cmd.Args)
			return nil, nil
		})

		cnf, err := LoadConfig(nil, "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cnf.CommitterName != "Jane Doe" {
			t.Errorf("expected committer identity to still be discovered, got %q", cnf.CommitterName)
		}
		if cnf.RemoteName != "" || cnf.RemoteURL != "" {
			t.Errorf("expected empty remote, got %q/%q", cnf.RemoteName, cnf.RemoteURL)
		}
	})

	t.Run("no-op when git is not installed", func(t *testing.T) {
		withFakeExecRaw(t, func(cmd *exec.Cmd) ([]byte, error) {
			t.Fatal("expected no exec calls when git isn't on PATH")
			return nil, nil
		})
		t.Setenv("PATH", "")

		cnf, err := LoadConfig(nil, "/repo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cnf != (Config{}) {
			t.Errorf("expected a zero-value Config, got %+v", cnf)
		}
	})
}
