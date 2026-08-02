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

	Agent        AgentConfig        `key:"agent"`
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

// AgentConfig points at the files a repository uses to describe itself to
// coding agents. It deliberately carries nothing but a toggle and paths:
// preferences are flattened to map[string]string before validation, so a
// nested list of skill or tool definitions cannot be expressed in this schema
// at all. The structured definitions live in the files these keys point at and
// are parsed by whatever consumes them, not here.
//
// See https://github.com/happy-sdk/.github/blob/main/spec/agent-manifest.md
type AgentConfig struct {
	Enabled settings.Bool `key:"enabled,save" default:"false"`
	// Instructions is a Markdown file of repository-specific agent
	// instructions, overriding any workspace-wide equivalent.
	Instructions settings.String `key:"instructions,save" default:".happy/AGENTS.md"`
	// Skills is a directory of <name>/SKILL.md procedures.
	Skills settings.String `key:"skills,save" default:".happy/skills"`
	// MCP declares the tools the repository exposes.
	MCP settings.String `key:"mcp,save" default:".happy/mcp.yaml"`
}

// Blueprint sets Enabled to true unless a saved preference already overrode it -
// settings.Bool fields may only declare a "false" struct-tag default, so "on by
// default" has to be established here in code instead, the same way
// ReleaserVerifyConfig.Blueprint does.
//
// On by default because the presence of the files above is already the opt-in
// signal: a repository that ships none of them contributes nothing regardless.
// The toggle exists so a repository that does ship them can still withhold
// them.
func (c *AgentConfig) Blueprint() (*settings.Blueprint, error) {
	c.Enabled = true
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
	Dist   settings.String      `key:"dist,save" default:".happy/build"`
	Bump   ReleaserBumpConfig   `key:"bump"`
	Verify ReleaserVerifyConfig `key:"verify"`
}

func (c *ReleaserConfig) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

// ReleaserBumpConfig controls how a breaking change or a Go version sync
// (see gomodule.Package.SyncGoVersion) advances a package's version -
// letting a project follow a compatibility policy other than plain semver,
// e.g. jumping straight to the next full-hundred minor instead of bumping
// major (see gomodule.BumpKind/BumpStrategy for the exact semantics).
type ReleaserBumpConfig struct {
	// Kind is "major" or "minor": which version component a breaking
	// change or Go version sync bumps.
	Kind settings.String `key:"kind,save" default:"major"`
	// Strategy is "single" (+1), "hundred" (next full x00), or "thousand"
	// (next full x000): how far Kind's component advances.
	Strategy settings.String `key:"strategy,save" default:"single"`
}

func (c *ReleaserBumpConfig) Blueprint() (*settings.Blueprint, error) {
	return settings.New(c)
}

// ReleaserVerifyConfig controls the release pipeline's final sanity check:
// after tagging and pushing, re-build and re-test each released module with
// go.work disabled, so its own go.mod is resolved against the tags that
// were just pushed instead of the on-disk workspace source every earlier
// step (lint, test, go mod tidy) resolves through. go.work makes every
// module in a monorepo satisfy its sibling imports from local source
// regardless of what its own go.mod actually requires, so nothing earlier
// in Release proves a downstream consumer - or this repo's own modules,
// used outside the workspace - can actually build against what was just
// tagged. Enabled by default: the point of the release command succeeding
// is that the maintainer can trust the published versions actually work
// together.
type ReleaserVerifyConfig struct {
	Enabled settings.Bool `key:"enabled,save" default:"false"`
}

// Blueprint sets Enabled to true unless a saved preference already overrode
// it - settings.Bool fields may only declare a "false" struct-tag default,
// so "on by default" has to be established here in code instead, the same
// way TestsConfig.Blueprint auto-detects a "go" binary.
func (c *ReleaserVerifyConfig) Blueprint() (*settings.Blueprint, error) {
	c.Enabled = true
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
