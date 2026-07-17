// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/happy-sdk/happy/addons/devel/pkg/changelog"
	"github.com/happy-sdk/happy/addons/devel/pkg/gitutils"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
)

type Config struct {
	Version version.Setting `key:"version,save" default:"v1.0.0"`

	// Ignore defines project-relative paths that should be skipped by
	// tooling such as lint and test. This mirrors the top-level
	// "ignore" list in .happy.yaml.
	Ignore settings.StringSlice `key:"ignore,save"`

	Changelog    changelog.Config   `key:"changelog"`
	Git          GitConfig          `key:"git"`
	Linter       LinterConfig       `key:"linter"`
	Releaser     ReleaserConfig     `key:"releaser"`
	Tests        TestsConfig        `key:"tests"`
	Dependencies DependenciesConfig `key:"deps"`
}

func (c *Config) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

type GitConfig struct {
	CommitterName  settings.String `key:"committer.name,save"`
	CommitterEmail settings.String `key:"committer.email,save"`
	AuthorName     settings.String `key:"author.name,save"`
	AuthorEmail    settings.String `key:"author.email,save"`
	Branch         settings.String `key:"branch,save" default:"main"`
	RemoteName     settings.String `key:"remote.name,save" default:"origin"`
	RemoteURL      settings.String `key:"remote.url,save"`
}

func (c *GitConfig) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

func newGitConfig(sess *session.Context, dir string) (gitcnf GitConfig, err error) {
	gitbin, err := exec.LookPath("git")
	if err != nil {
		return gitcnf, nil
	}
	// Get committer name
	committerCmd := exec.Command(gitbin, "config", "user.name")
	committerCmd.Dir = dir
	committer, err := cli.Exec(sess, committerCmd)
	if err != nil {
		return
	}
	gitcnf.CommitterName = settings.String(committer)
	gitcnf.AuthorName = gitcnf.CommitterName

	// Get committer email
	emailCmd := exec.Command(gitbin, "config", "user.email")
	emailCmd.Dir = dir
	email, err := cli.Exec(sess, emailCmd)
	if err != nil {
		return
	}
	gitcnf.CommitterEmail = settings.String(email)
	gitcnf.AuthorEmail = gitcnf.CommitterEmail

	// Get current branch
	branch, err := gitutils.CurrentBranch(sess, dir)
	if err != nil {
		return
	}
	gitcnf.Branch = settings.String(branch)

	// Get remote
	remoteName, remoteURL, err := gitutils.CurrentRemote(sess, dir)
	if err != nil {
		return
	}
	gitcnf.RemoteName = settings.String(remoteName)
	gitcnf.RemoteURL = settings.String(remoteURL)

	return
}

type LinterConfig struct {
	Enabled      settings.Bool            `key:"enabled,save" default:"false"`
	GolangCILint LinterGolangCILintConfig `key:"golangci-lint"`
}

func (c *LinterConfig) Blueprint() (*settings.Blueprint, error) {
	_, err := settings.New(&c.GolangCILint)
	if err != nil {
		return nil, err
	}
	if c.GolangCILint.Enabled {
		c.Enabled = true
	}
	bp, err := settings.New(c)
	if err != nil {
		return nil, err
	}
	return bp, nil
}

type LinterGolangCILintConfig struct {
	Enabled settings.Bool   `key:"enabled,save" default:"false"`
	Path    settings.String `key:"path,save" default:""`
}

func (c *LinterGolangCILintConfig) Blueprint() (*settings.Blueprint, error) {
	golangciLint, err := exec.LookPath("golangci-lint")
	if err == nil {
		if golangciLint != "" {
			c.Path = settings.String(golangciLint)
		}
		if c.Path != "" {
			c.Enabled = true
		}
	} else {
		gloangciLintBinGh, isSet := os.LookupEnv("GITHUB_WORKSPACE")
		if isSet {
			c.Path = settings.String(filepath.Join(gloangciLintBinGh, "bin", "golangci-lint"))
			c.Enabled = true
		}
	}

	return settings.New(c)
}

type TestsConfig struct {
	Enabled settings.Bool `key:"enabled,save" default:"false"`
}

func (t *TestsConfig) Blueprint() (*settings.Blueprint, error) {
	_, err := exec.LookPath("go")
	if err == nil {
		t.Enabled = true
	}
	return settings.New(t)
}

type ReleaserConfig struct {
	Enabled settings.Bool   `key:"enabled,save" default:"false"`
	Dist    settings.String `key:"dist,save" default:"dist"`
}

func (c *ReleaserConfig) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

type DependenciesConfig struct {
	Latest DependenciesGoConfig `key:"go"`
}

func (c *DependenciesConfig) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

type DependenciesGoConfig struct {
	Latest settings.StringSlice `key:"latest"`
}

func (c *DependenciesGoConfig) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}
