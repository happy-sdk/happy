// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
)

var Error = errors.New("git")

// IsGitRepo checks if the given directory is a Git repository.
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil || !os.IsNotExist(err)
}

func NewConfig() (*options.Spec, error) {
	return options.New("git",
		options.NewOption("repo.found", false),
		options.NewOption("repo.root", "").
			Validator(func(opt options.Option) error {
				repoRoot := opt.Value().String()
				if repoRoot == "" {
					return nil
				}
				if !IsGitRepo(repoRoot) {
					return fmt.Errorf("not a valid Git repository: %s", repoRoot)
				}
				return nil
			}),
		options.NewOption("repo.branch", ""),
		options.NewOption("repo.remote.name", ""),
		options.NewOption("repo.remote.url", ""),
		options.NewOption("repo.dirty", ""),
		options.NewOption("committer.name", ""),
		options.NewOption("committer.email", ""),
	)
}

func DetectGitRepo(sess *session.Context, config *options.Options) error {
	if !config.Get("local.wd").IsSet() {
		return fmt.Errorf("%w: local.wd is not set", Error)
	}
	dir := config.Get("local.wd").String()
	for {
		// Check if the current path is a Git repository
		if IsGitRepo(dir) {
			if err := config.Set("git.repo.found", true); err != nil {
				return err
			}
			if err := config.Set("git.repo.root", dir); err != nil {
				return err
			}
			break
		}
		parent := filepath.Dir(dir)
		// Check if we've reached the root directory
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil
}

func LoadInfo(sess *session.Context, config *options.Options) error {

	if !config.Get("git.repo.found").Variable().Bool() {
		return nil
	}

	repoRoot := config.Get("git.repo.root").String()

	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoRoot
	branch, err := cli.ExecRaw(sess, branchCmd)
	if err != nil {
		return err
	}
	if err := config.Set("git.repo.branch", strings.TrimSpace(string(branch))); err != nil {
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
		if err := config.Set("git.repo.remote.name", strings.TrimSpace(remoteParts[0])); err != nil {
			return err
		}
	}

	// Get origin URL
	remoteConfigKey := fmt.Sprintf("remote.%s.url", config.Get("git.repo.remote.name").String())
	remoteURLCmd := exec.Command("git", "config", "--get", remoteConfigKey)
	remoteURLCmd.Dir = repoRoot
	remoteURL, err := cli.ExecRaw(sess, remoteURLCmd)
	if err != nil {
		return err
	}
	if err := config.Set("git.repo.remote.url", strings.TrimSpace(string(remoteURL))); err != nil {
		return err
	}

	// Check for uncommitted changes
	dirty := Dirty(sess, repoRoot, ".")
	if err := config.Set("git.repo.dirty", dirty); err != nil {
		return err
	}

	// Get committer name
	committerCmd := exec.Command("git", "config", "user.name")
	committerCmd.Dir = repoRoot
	committer, err := cli.ExecRaw(sess, committerCmd)
	if err != nil {
		return err
	}
	if err := config.Set("git.committer.name", strings.TrimSpace(string(committer))); err != nil {
		return err
	}

	// Get committer email
	emailCmd := exec.Command("git", "config", "user.email")
	emailCmd.Dir = repoRoot
	email, err := cli.ExecRaw(sess, emailCmd)
	if err != nil {
		return err
	}
	if err := config.Set("git.committer.email", strings.TrimSpace(string(email))); err != nil {
		return err
	}

	return nil
}

func TagExists(sess *session.Context, wd string, tag string) bool {
	tagCmd := exec.Command("git", "tag", "-l", tag)
	tagCmd.Dir = wd
	tagOutput, err := cli.ExecRaw(sess, tagCmd)
	if err != nil {
		return false
	}
	return strings.Contains(string(tagOutput), tag)
}

func RemoteTagExists(sess *session.Context, wd string, origin, tag string) bool {
	tagCmd := exec.Command("git", "ls-remote", "--tags", origin, tag)
	tagCmd.Dir = wd
	tagOutput, err := cli.ExecRaw(sess, tagCmd)
	if err != nil {
		return false
	}
	return strings.Contains(string(tagOutput), tag)
}

func Dirty(sess *session.Context, wd string, path string) bool {
	statusCmd := exec.Command("git", "status", "--porcelain", path)
	statusCmd.Dir = wd
	status, err := cli.ExecRaw(sess, statusCmd)
	if err != nil {
		return false
	}
	return bytes.TrimSpace(status) != nil
}

func Commit(sess *session.Context, wd, spath, commitMsg string) error {
	if !Dirty(sess, wd, spath) {
		return nil
	}

	gitadd := exec.Command("git", "add", spath)
	gitadd.Dir = wd
	if err := cli.Run(sess, gitadd); err != nil {
		return err
	}

	gitcommit := exec.Command("git", "commit", "-sm", commitMsg)
	gitcommit.Dir = wd
	if err := cli.Run(sess, gitcommit); err != nil {
		return err
	}

	return nil
}
