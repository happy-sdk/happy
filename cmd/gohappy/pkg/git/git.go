// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
)

// IsGitRepo checks if the given directory is a Git repository.
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil || !os.IsNotExist(err)
}

func LoadInfo(sess *session.Context, m *vars.Map, dir string) error {
	for {
		// Check if the current path is a Git repository
		if IsGitRepo(dir) {
			if err := m.Store("git.repo.found", true); err != nil {
				return err
			}
			if err := m.Store("git.repo.root", dir); err != nil {
				return err
			}
		}

		parent := filepath.Dir(dir)
		// Check if we've reached the root directory
		if parent == dir {
			break
		}
		dir = parent
	}

	if !m.Get("git.repo.found").Bool() {
		return nil
	}

	repoRoot := m.Get("git.repo.root").String()

	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoRoot
	branch, err := cli.ExecRaw(sess, branchCmd)
	if err != nil {
		return err
	}
	if err := m.Store("git.repo.branch", strings.TrimSpace(string(branch))); err != nil {
		return err
	}

	// Get remote name
	remoteCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	remoteCmd.Dir = repoRoot
	remote, err := cli.ExecRaw(sess, remoteCmd)
	if err != nil {
		return err
	}
	remoteParts := strings.SplitN(strings.TrimSpace(string(remote)), "/", 2)
	if len(remoteParts) > 0 {
		if err := m.Store("git.repo.remote.name", strings.TrimSpace(remoteParts[0])); err != nil {
			return err
		}
	}

	// Get origin URL
	remoteConfigKey := fmt.Sprintf("remote.%s.url", m.Get("git.repo.remote.name").String())
	remoteURLCmd := exec.Command("git", "config", "--get", remoteConfigKey)
	remoteURLCmd.Dir = repoRoot
	remoteURL, err := cli.ExecRaw(sess, remoteURLCmd)
	if err != nil {
		return err
	}
	if err := m.Store("git.repo.remote.url", strings.TrimSpace(string(remoteURL))); err != nil {
		return err
	}

	// Check for uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoRoot
	status, err := cli.ExecRaw(sess, statusCmd)
	if err != nil {
		return err
	}
	dirty := bytes.TrimSpace(status) != nil
	if err := m.Store("git.repo.dirty", dirty); err != nil {
		return err
	}

	// Get committer name
	committerCmd := exec.Command("git", "config", "user.name")
	committerCmd.Dir = repoRoot
	committer, err := cli.ExecRaw(sess, committerCmd)
	if err != nil {
		return err
	}
	if err := m.Store("git.committer.name", strings.TrimSpace(string(committer))); err != nil {
		return err
	}

	// Get committer email
	emailCmd := exec.Command("git", "config", "user.email")
	emailCmd.Dir = repoRoot
	email, err := cli.ExecRaw(sess, emailCmd)
	if err != nil {
		return err
	}
	if err := m.Store("git.committer.email", strings.TrimSpace(string(email))); err != nil {
		return err
	}

	return nil
}
