// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/happy-sdk/happy/lib/scm/gitutils"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/version"
)

type Config struct {
	Version version.Setting `key:"version,save" default:"v1.0.0"`

	// Ignore defines project-relative paths that should be skipped by
	// tooling such as lint and test. This mirrors the top-level
	// "ignore" list in .happy.yaml.
	Ignore settings.StringSlice `key:"ignore,save"`

	Changelog    ChangelogConfig    `key:"changelog"`
	Git          gitutils.Config    `key:"git"`
	Linter       LinterConfig       `key:"linter"`
	Releaser     ReleaserConfig     `key:"releaser"`
	Tests        TestsConfig        `key:"tests"`
	Dependencies DependenciesConfig `key:"deps"`
}

func (c *Config) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
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
	// Disable lists linter/analyzer names to pass to golangci-lint's
	// --disable flag. Meant for temporary workarounds (e.g. an upstream
	// analyzer that's currently broken on a given toolchain), not permanent
	// project lint policy.
	Disable settings.StringSlice `key:"disable,save"`
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

// ChangelogConfig toggles whether the releaser generates a changelog at
// all. It's a project-level release-pipeline setting, not something the
// reusable lib/changelog package itself needs to know about.
type ChangelogConfig struct {
	Disabled settings.Bool `key:"disabled,save" default:"false"`
}

func (c *ChangelogConfig) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

type ReleaserConfig struct {
	Enabled settings.Bool `key:"enabled,save" default:"false"`
	// Dist is the releaser's working/build directory, relative to the
	// project root - historically a plain "dist" dir, now living under the
	// shared ".happy" directory alongside happyctl's other own files.
	Dist settings.String `key:"dist,save" default:".happy/build"`
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
