// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package gitutils

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
)

var Error = errors.New("git")

// IsRepository checks if the given directory is a Git repository.
func IsRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil || !os.IsNotExist(err)
}

// FindRepositoryRoot locates the root directory of the Git repository containing wd.
// It returns:
//   - dir: the absolute path to the repository root (or the original wd if none found)
//   - found: true if a ".git" folder was discovered, false otherwise
//   - err: any error encountered resolving the absolute path of wd
//
// Starting at wd, this function ascends parent directories until it finds a
// ".git" directory. If found, it returns that directory and found=true.
// If no repository is detected, it returns the original wd and found=false.
func FindRepositoryRoot(wd string) (dir string, found bool, err error) {
	dir, err = filepath.Abs(wd)
	if err != nil {
		return wd, false, err
	}
	for {
		if IsRepository(dir) {
			return dir, true, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir || dir == "/" || dir == "." {
			break
		}
		dir = parent
	}
	return wd, false, nil
}

func NewIgnoreMatcher(patterns []string, domain []string) gitignore.Matcher {
	var ps []gitignore.Pattern
	for _, p := range patterns {
		pat := gitignore.ParsePattern(p, domain)
		ps = append(ps, pat)
	}

	return gitignore.NewMatcher(ps)
}

func NewConfig() (*options.Spec, error) {
	return options.New("git",
		options.NewOption("repo.found", false),
		options.NewOption("loaded", false),
		options.NewOption("repo.root", "").
			Validator(func(opt options.Option) error {
				dir := opt.Value().String()
				if dir == "" {
					return nil
				}
				if !IsRepository(dir) {
					return fmt.Errorf("not a valid Git repository: %s", dir)
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

func Dirty(sess *session.Context, wd string, path string) bool {
	statusCmd := exec.Command("git", "status", "--porcelain", path)
	statusCmd.Dir = wd
	status, err := cli.ExecRaw(sess, statusCmd)
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(status)) > 0
}

func CurrentBranch(sess *session.Context, wd string) (string, error) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = wd
	branch, err := cli.ExecRaw(sess, branchCmd)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(branch)), nil
}

func CurrentRemote(sess *session.Context, wd string) (name, url string, err error) {
	// Get remote name
	remoteNameCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	remoteNameCmd.Dir = wd
	remoteName, err := cli.ExecRaw(sess, remoteNameCmd)
	if err != nil {
		return
	}
	remoteNameParts := strings.SplitN(strings.TrimSpace(string(remoteName)), "/", 2)
	if len(remoteNameParts) > 0 {
		name = strings.TrimSpace(remoteNameParts[0])
	}

	// Get origin URL
	remoteConfigKey := fmt.Sprintf("remote.%s.url", name)
	remoteURLCmd := exec.Command("git", "config", "--get", remoteConfigKey)
	remoteURLCmd.Dir = wd
	remoteURL, err := cli.ExecRaw(sess, remoteURLCmd)
	if err != nil {
		return
	}
	url = strings.TrimSpace(string(remoteURL))

	return
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

func TagExists(sess *session.Context, wd string, tag string) bool {
	tagCmd := exec.Command("git", "tag", "-l", tag)
	tagCmd.Dir = wd
	tagOutput, err := cli.ExecRaw(sess, tagCmd)
	if err != nil {
		return false
	}
	return strings.Contains(string(tagOutput), tag)
}

func Commit(sess *session.Context, wd string, arg []string, commitMsg string) error {
	if !Dirty(sess, wd, ".") {
		return nil
	}
	gargs := []string{"add"}
	gargs = append(gargs, arg...)

	gitadd := exec.Command("git", gargs...)
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

func Tag(sess *session.Context, wd, tag, message string) error {
	gitTag := exec.Command("git", "tag", "-s", tag, "-m", message)
	gitTag.Dir = wd
	if err := cli.Run(sess, gitTag); err != nil {
		return err
	}

	return nil
}
