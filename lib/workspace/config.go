// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FileName is the workspace marker, deliberately distinct from the project
// manifest. A workspace root must never be mistaken for a project: project
// detection keys on ".happy.yaml", and a marker sharing that name would make
// every workspace root look like a project to happyctl.
const FileName = ".happy-workspace.yaml"

// Version is the only marker schema version understood.
const Version = "1"

// RepoPlaceholder is substituted with a repository name in Org.Remote.
const RepoPlaceholder = "{repo}"

// Default layout, used when the marker omits them.
const (
	DefaultRepos   = "src"
	DefaultScratch = "workspace"
)

// Config is the contents of the workspace marker.
//
// Everything except Version is optional: a marker containing only a version
// yields the recommended layout. It is parsed as plain YAML rather than through
// pkg/settings because Repos is a list of objects, which the settings schema
// cannot express - preferences are flattened to map[string]string before
// validation, so only scalars and string slices survive.
type Config struct {
	Version string `yaml:"version"`
	Org     Org    `yaml:"org,omitempty"`
	Layout  Layout `yaml:"layout,omitempty"`
	// Repos are the repositories this workspace expects. It is optional and
	// advisory: checkouts present on disk are discovered whether or not they
	// are listed, and listing one that is absent is what lets clone and doctor
	// say something useful about it.
	Repos []Repo `yaml:"repos,omitempty"`
}

// Org identifies where repositories come from by default. It is optional, and
// a workspace may span several hosts by giving repositories their own remotes.
type Org struct {
	Name string `yaml:"name,omitempty"`
	// Remote is a template containing {repo}, e.g.
	// git@github.com:happy-sdk/{repo}.git
	Remote string `yaml:"remote,omitempty"`
}

// Layout is where things live inside the workspace, relative to its root.
type Layout struct {
	// Repos holds checkouts. Defaults to "src".
	Repos string `yaml:"repos,omitempty"`
	// Scratch holds the developer's own notes and files. Defaults to
	// "workspace"; empty means the workspace has none and none is created.
	Scratch string `yaml:"scratch,omitempty"`
}

// Repo is one expected repository.
type Repo struct {
	// Name is the repository name, used to fill Org.Remote.
	Name string `yaml:"name"`
	// Dir overrides the local directory name. Without it the directory is
	// Name, which is usually right but not always: a repository literally
	// called ".github" would otherwise become a hidden directory.
	Dir string `yaml:"dir,omitempty"`
	// Remote overrides Org.Remote for this repository, letting one workspace
	// span organizations or hosts.
	Remote string `yaml:"remote,omitempty"`
}

// LocalDir is the directory name this repository is checked out as.
func (r Repo) LocalDir() string {
	if r.Dir != "" {
		return r.Dir
	}
	return r.Name
}

// Default returns the recommended configuration, which is what an
// initialization wizard writes unless told otherwise.
func Default() Config {
	return Config{
		Version: Version,
		Layout: Layout{
			Repos:   DefaultRepos,
			Scratch: DefaultScratch,
		},
	}
}

// withDefaults fills in omitted layout values. Scratch is left alone: empty
// means "no scratch directory", which is a choice rather than an omission, and
// cannot be distinguished from unset in YAML.
func (c Config) withDefaults() Config {
	if c.Layout.Repos == "" {
		c.Layout.Repos = DefaultRepos
	}
	return c
}

// Validate reports whether the configuration is usable.
func (c Config) Validate() error {
	if c.Version != Version {
		return fmt.Errorf("%w: unsupported version %q, want %q", ErrConfig, c.Version, Version)
	}
	if err := validRelDir("layout.repos", c.Layout.Repos); err != nil {
		return err
	}
	if c.Layout.Scratch != "" {
		if err := validRelDir("layout.scratch", c.Layout.Scratch); err != nil {
			return err
		}
		if filepath.Clean(c.Layout.Scratch) == filepath.Clean(c.Layout.Repos) {
			return fmt.Errorf("%w: layout.scratch and layout.repos are both %q", ErrConfig, c.Layout.Scratch)
		}
	}
	if c.Org.Remote != "" && !strings.Contains(c.Org.Remote, RepoPlaceholder) {
		return fmt.Errorf("%w: org.remote %q must contain %s", ErrConfig, c.Org.Remote, RepoPlaceholder)
	}

	seenName := make(map[string]bool, len(c.Repos))
	seenDir := make(map[string]bool, len(c.Repos))
	for i, r := range c.Repos {
		if r.Name == "" {
			return fmt.Errorf("%w: repos[%d] has no name", ErrConfig, i)
		}
		if seenName[r.Name] {
			return fmt.Errorf("%w: duplicate repository %q", ErrConfig, r.Name)
		}
		seenName[r.Name] = true

		dir := r.LocalDir()
		if err := validRelDir(fmt.Sprintf("repos[%d].dir", i), dir); err != nil {
			return err
		}
		if strings.ContainsRune(dir, '/') || strings.ContainsRune(dir, filepath.Separator) {
			return fmt.Errorf("%w: repos[%d] directory %q must be a single path element", ErrConfig, i, dir)
		}
		if seenDir[dir] {
			return fmt.Errorf("%w: two repositories both check out as %q", ErrConfig, dir)
		}
		seenDir[dir] = true

		if r.Remote == "" && c.Org.Remote == "" {
			return fmt.Errorf("%w: repository %q has no remote and org.remote is unset", ErrConfig, r.Name)
		}
	}
	return nil
}

// RemoteFor resolves the clone URL for a repository: its own remote if it has
// one, otherwise the organization template with {repo} substituted.
func (c Config) RemoteFor(name string) (string, error) {
	for _, r := range c.Repos {
		if r.Name != name {
			continue
		}
		if r.Remote != "" {
			return r.Remote, nil
		}
		break
	}
	if c.Org.Remote == "" {
		return "", fmt.Errorf("%w: no remote for %q and org.remote is unset", ErrConfig, name)
	}
	return strings.ReplaceAll(c.Org.Remote, RepoPlaceholder, name), nil
}

// Repo returns the declared repository with the given name.
func (c Config) Repo(name string) (Repo, bool) {
	for _, r := range c.Repos {
		if r.Name == name {
			return r, true
		}
	}
	return Repo{}, false
}

// validRelDir rejects anything that would place a workspace directory outside
// the workspace root.
func validRelDir(field, dir string) error {
	if dir == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrConfig, field)
	}
	if filepath.IsAbs(dir) {
		return fmt.Errorf("%w: %s %q must be relative to the workspace root", ErrConfig, field, dir)
	}
	clean := filepath.Clean(dir)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == "." {
		return fmt.Errorf("%w: %s %q must stay inside the workspace root", ErrConfig, field, dir)
	}
	return nil
}
