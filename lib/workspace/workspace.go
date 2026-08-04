// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

// Package workspace resolves and describes a development workspace: a
// directory holding several repository checkouts side by side, plus the
// developer's own scratch space.
//
// A workspace is local state, not a repository. It is marked by a
// .happy-workspace.yaml file rather than by being a git checkout, which keeps
// it from being mistaken for a project and keeps checkouts from being nested
// inside another repository - nesting breaks tools that resolve a project root
// by ascending, and does so silently.
//
// Nothing here is specific to any organization. The marker names where
// repositories come from, so the same tooling serves any set of repositories
// developed together.
//
//	<root>/
//	├── .happy-workspace.yaml
//	├── src/          repository checkouts   (layout.repos)
//	└── workspace/    personal scratch space (layout.scratch, optional)
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

var (
	Error       = errors.New("workspace")
	ErrConfig   = fmt.Errorf("%w: config", Error)
	ErrNotFound = fmt.Errorf("%w: not found", Error)
	ErrExists   = fmt.Errorf("%w: already exists", Error)
)

// EnvRoot names the workspace root when set. Tools launched by another process
// - an editor, or an MCP client - start in an unpredictable working directory,
// so an explicit root matters more than the upward search does.
const EnvRoot = "HAPPY_WORKSPACE"

// Workspace is a resolved workspace root and its configuration.
type Workspace struct {
	// Root is the absolute path to the workspace root.
	Root string
	// Config is the marker's contents, with defaults applied.
	Config Config
}

// Checkout is a repository present on disk under the workspace's repos
// directory.
type Checkout struct {
	// Name is the directory name, which is how the checkout is addressed.
	Name string
	// Dir is the absolute path to the checkout.
	Dir string
	// Repo is the matching declaration from the marker, if the workspace
	// declares one.
	Repo *Repo
	// IsGit reports whether the directory is a git checkout.
	IsGit bool
}

// Declared reports whether the marker lists this checkout.
func (c Checkout) Declared() bool { return c.Repo != nil }

// Find ascends from start looking for a workspace marker.
func Find(start string) (*Workspace, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, FileName)); err == nil {
			return Load(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, fmt.Errorf("%w: no %s from %s upward", ErrNotFound, FileName, start)
		}
		dir = parent
	}
}

// Resolve picks a workspace from an explicit root, then the environment, then
// by ascending from the working directory.
func Resolve(explicit string) (*Workspace, error) {
	if explicit != "" {
		return Load(explicit)
	}
	if env := strings.TrimSpace(os.Getenv(EnvRoot)); env != "" {
		return Load(env)
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return Find(wd)
}

// Load reads the marker in root.
func Load(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(abs, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s has no %s", ErrNotFound, abs, FileName)
		}
		return nil, fmt.Errorf("%w: reading %s: %s", Error, path, err.Error())
	}

	var cnf Config
	if err := yaml.Unmarshal(data, &cnf); err != nil {
		return nil, fmt.Errorf("%w: parsing %s: %s", ErrConfig, path, err.Error())
	}
	cnf = cnf.withDefaults()
	if err := cnf.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &Workspace{Root: abs, Config: cnf}, nil
}

// Create writes a marker and the directories it describes. It refuses to
// overwrite an existing marker, since that would silently discard whatever the
// workspace was configured to be.
func Create(root string, cnf Config) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(abs, FileName)); err == nil {
		return nil, fmt.Errorf("%w: %s already has %s", ErrExists, abs, FileName)
	}

	cnf = cnf.withDefaults()
	if err := cnf.Validate(); err != nil {
		return nil, err
	}

	ws := &Workspace{Root: abs, Config: cnf}
	if err := os.MkdirAll(ws.ReposDir(), 0o755); err != nil {
		return nil, err
	}
	if scratch := ws.ScratchDir(); scratch != "" {
		if err := os.MkdirAll(scratch, 0o755); err != nil {
			return nil, err
		}
	}
	if err := ws.Save(); err != nil {
		return nil, err
	}
	return ws, nil
}

// Save writes the marker back to disk.
func (w *Workspace) Save() error {
	data, err := yaml.Marshal(w.Config)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrConfig, err.Error())
	}
	return os.WriteFile(w.MarkerPath(), data, 0o600)
}

// MarkerPath is the absolute path to the workspace marker.
func (w *Workspace) MarkerPath() string { return filepath.Join(w.Root, FileName) }

// ReposDir is where checkouts live.
func (w *Workspace) ReposDir() string {
	return filepath.Join(w.Root, filepath.FromSlash(w.Config.Layout.Repos))
}

// ScratchDir is the developer's own space, or empty when the workspace has
// none.
func (w *Workspace) ScratchDir() string {
	if w.Config.Layout.Scratch == "" {
		return ""
	}
	return filepath.Join(w.Root, filepath.FromSlash(w.Config.Layout.Scratch))
}

// Dir is where the named repository is or would be checked out. The name is
// the repository name; a marker entry may map it to a different directory.
func (w *Workspace) Dir(name string) string {
	if r, ok := w.Config.Repo(name); ok {
		return filepath.Join(w.ReposDir(), r.LocalDir())
	}
	return filepath.Join(w.ReposDir(), name)
}

// Checkouts lists what is present under the repos directory, in stable order.
//
// Directories are reported whether or not the marker declares them and whether
// or not they are git checkouts, because "present but not a checkout" and
// "declared but absent" are both things a caller needs to be able to say.
// Hidden directories are included: a repository may legitimately be named
// ".github", and the marker's dir override exists precisely so it need not be.
func (w *Workspace) Checkouts() ([]Checkout, error) {
	entries, err := os.ReadDir(w.ReposDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: reading %s: %s", Error, w.ReposDir(), err.Error())
	}

	byDir := make(map[string]*Repo, len(w.Config.Repos))
	for i := range w.Config.Repos {
		r := &w.Config.Repos[i]
		byDir[r.LocalDir()] = r
	}

	var out []Checkout
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(w.ReposDir(), e.Name())
		out = append(out, Checkout{
			Name:  e.Name(),
			Dir:   dir,
			Repo:  byDir[e.Name()],
			IsGit: isGitCheckout(dir),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Missing lists declared repositories that are not checked out, which is what
// a clone or doctor command reports.
func (w *Workspace) Missing() ([]Repo, error) {
	checkouts, err := w.Checkouts()
	if err != nil {
		return nil, err
	}
	present := make(map[string]bool, len(checkouts))
	for _, c := range checkouts {
		present[c.Name] = true
	}

	var missing []Repo
	for _, r := range w.Config.Repos {
		if !present[r.LocalDir()] {
			missing = append(missing, r)
		}
	}
	return missing, nil
}

// isGitCheckout reports whether dir is a git working tree root. The .git entry
// is a directory in a normal clone and a file in a worktree or submodule.
func isGitCheckout(dir string) bool {
	fi, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	return fi.IsDir() || fi.Mode().IsRegular()
}
