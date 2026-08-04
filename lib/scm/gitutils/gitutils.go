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

	git "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/format/gitignore"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
)

var Error = errors.New("gitutils")

// execRaw and execRun are indirection points so tests can substitute a fake
// command runner instead of actually invoking `git` - cli.ExecRaw/cli.Run
// require a real, fully-booted *session.Context internally (for logging and
// context cancellation), which nothing in this repo constructs standalone
// in tests. Substituting these lets tests exercise this package's own
// argument-building and output-parsing logic deterministically, without a
// real session or a real git binary.
var (
	execRaw = cli.ExecRaw
	execRun = cli.Run
)

// IsRepository reports whether path is a valid Git repository (bare or
// with a worktree). It opens the repository via go-git rather than merely
// checking for a ".git" entry, so a directory with an empty or otherwise
// malformed ".git" (e.g. missing HEAD/objects/refs) correctly reports
// false instead of a false positive.
func IsRepository(path string) bool {
	_, err := git.PlainOpen(path)
	return err == nil
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
		if parent == dir {
			// filepath.Abs already made dir absolute, so the only way
			// ascending stops making progress is at the filesystem root.
			break
		}
		dir = parent
	}
	return wd, false, nil
}

// NewIgnoreMatcher builds a gitignore.Matcher from patterns (in .gitignore
// syntax), scoped under domain - the slash-separated path components the
// patterns are relative to (nil for the repository root).
func NewIgnoreMatcher(patterns []string, domain []string) gitignore.Matcher {
	var ps []gitignore.Pattern
	for _, p := range patterns {
		pat := gitignore.ParsePattern(p, domain)
		ps = append(ps, pat)
	}

	return gitignore.NewMatcher(ps)
}

func Dirty(sess *session.Context, wd string, path string) bool {
	statusCmd := exec.Command("git", "status", "--porcelain", path)
	statusCmd.Dir = wd
	status, err := execRaw(sess, statusCmd)
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(status)) > 0
}

func CurrentBranch(sess *session.Context, wd string) (string, error) {
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = wd
	branch, err := execRaw(sess, branchCmd)
	if err != nil {
		return "", fmt.Errorf("%w: current branch: %w", Error, err)
	}

	return strings.TrimSpace(string(branch)), nil
}

func CurrentRemote(sess *session.Context, wd string) (name, url string, err error) {
	// Get remote name
	remoteNameCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	remoteNameCmd.Dir = wd
	remoteName, err := execRaw(sess, remoteNameCmd)
	if err != nil {
		err = fmt.Errorf("%w: current remote: %w", Error, err)
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
	remoteURL, err := execRaw(sess, remoteURLCmd)
	if err != nil {
		err = fmt.Errorf("%w: current remote url: %w", Error, err)
		return
	}
	url = strings.TrimSpace(string(remoteURL))

	return
}

// CommitterIdentity reads the configured git committer name and email for
// wd (i.e. `git config user.name`/`user.email`).
func CommitterIdentity(sess *session.Context, wd string) (name, email string, err error) {
	nameCmd := exec.Command("git", "config", "user.name")
	nameCmd.Dir = wd
	nameOut, err := execRaw(sess, nameCmd)
	if err != nil {
		return "", "", fmt.Errorf("%w: committer name: %w", Error, err)
	}
	name = strings.TrimSpace(string(nameOut))

	emailCmd := exec.Command("git", "config", "user.email")
	emailCmd.Dir = wd
	emailOut, err := execRaw(sess, emailCmd)
	if err != nil {
		return "", "", fmt.Errorf("%w: committer email: %w", Error, err)
	}
	email = strings.TrimSpace(string(emailOut))

	return name, email, nil
}

func RemoteTagExists(sess *session.Context, wd string, origin, tag string) bool {
	tagCmd := exec.Command("git", "ls-remote", "--tags", origin, tag)
	tagCmd.Dir = wd
	tagOutput, err := execRaw(sess, tagCmd)
	if err != nil {
		return false
	}
	return tagRefMatches(string(tagOutput), tag)
}

func TagExists(sess *session.Context, wd string, tag string) bool {
	tagCmd := exec.Command("git", "tag", "-l", tag)
	tagCmd.Dir = wd
	tagOutput, err := execRaw(sess, tagCmd)
	if err != nil {
		return false
	}
	return tagRefMatches(string(tagOutput), tag)
}

// tagRefMatches reports whether output (from `git tag -l <tag>`, which
// lists just the tag name per line, or `git ls-remote --tags <remote>
// <tag>`, which lists "<hash>\trefs/tags/<tag>" per line, with dereferenced
// annotated tags also getting a "^{}"-suffixed line) contains a line that
// refers to exactly tag - not merely a substring match, which would
// wrongly report e.g. "v1.2" as existing when only "v1.2.0" does.
func tagRefMatches(output, tag string) bool {
	want := "refs/tags/" + tag
	for line := range strings.Lines(output) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		ref := fields[len(fields)-1]
		if ref == tag || ref == want || ref == want+"^{}" {
			return true
		}
	}
	return false
}

func Commit(sess *session.Context, wd string, arg []string, commitMsg string) error {
	if !Dirty(sess, wd, ".") {
		return nil
	}
	gargs := []string{"add"}
	gargs = append(gargs, arg...)

	gitadd := exec.Command("git", gargs...)
	gitadd.Dir = wd
	if err := execRun(sess, gitadd); err != nil {
		return fmt.Errorf("%w: add: %w", Error, err)
	}

	gitcommit := exec.Command("git", "commit", "-sm", commitMsg)
	gitcommit.Dir = wd
	if err := execRun(sess, gitcommit); err != nil {
		return fmt.Errorf("%w: commit: %w", Error, err)
	}

	return nil
}

// Clone clones remote into dir, which must not already exist as a non-empty
// directory. The parent is created if needed, so a caller need not prepare the
// destination.
//
// Cloning is a network operation with side effects on disk, so it belongs
// behind an explicit command rather than anything that runs implicitly.
func Clone(sess *session.Context, remote, dir string) error {
	if remote == "" {
		return fmt.Errorf("%w: clone: no remote", Error)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("%w: clone: %w", Error, err)
	}
	gitClone := exec.Command("git", "clone", remote, dir)
	if err := execRun(sess, gitClone); err != nil {
		return fmt.Errorf("%w: clone %s: %w", Error, remote, err)
	}
	return nil
}

func Tag(sess *session.Context, wd, tag, message string) error {
	gitTag := exec.Command("git", "tag", "-s", tag, "-m", message)
	gitTag.Dir = wd
	if err := execRun(sess, gitTag); err != nil {
		return fmt.Errorf("%w: tag: %w", Error, err)
	}

	return nil
}
