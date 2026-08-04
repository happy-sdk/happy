// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

// Package discovery reads the agent manifests that repositories in a workspace
// ship under their .happy directory.
//
// Locating the workspace and enumerating its checkouts is lib/workspace's job;
// this package only interprets what each checkout declares. Nothing here knows
// about any specific repository, so a repository that adds a skill or a tool is
// picked up without a change to this code.
package discovery

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/happy-sdk/happy/lib/workspace"
)

var (
	Error           = errors.New("discovery")
	ErrInvalidValue = fmt.Errorf("%w: invalid value", Error)
)

// Default locations a repository uses when its project manifest says nothing.
const (
	DefaultInstructions = ".happy/AGENTS.md"
	DefaultSkills       = ".happy/skills"
	DefaultManifest     = ".happy/mcp.yaml"
)

// projectManifest is the repository's .happy.yaml, read leniently.
//
// happyctl validates that file against a strict schema where an unknown key is
// a hard error. Here only the agent section is wanted, so it is parsed on its
// own: a repository is not punished for carrying configuration this package
// has no opinion about, and a manifest this package cannot fully understand
// still yields its agent paths.
type projectManifest struct {
	Agent struct {
		Enabled      *bool  `yaml:"enabled"`
		Instructions string `yaml:"instructions"`
		Skills       string `yaml:"skills"`
		MCP          string `yaml:"mcp"`
	} `yaml:"agent"`
}

// Issue is a problem found while reading a repository's manifest. A repository
// with issues contributes nothing, but never stops another from loading - one
// bad file must not empty the whole server.
type Issue struct {
	Repo    string
	Path    string
	Message string
}

func (i Issue) String() string {
	if i.Path != "" {
		return fmt.Sprintf("%s: %s: %s", i.Repo, i.Path, i.Message)
	}
	return fmt.Sprintf("%s: %s", i.Repo, i.Message)
}

// Repo is one checkout and whatever agent context it declares.
type Repo struct {
	// Name is the checkout directory name.
	Name string
	// Dir is the absolute path to the checkout.
	Dir string
	// Namespace prefixes this repository's tools and skills.
	Namespace string
	// Description comes from the manifest.
	Description string
	// Instructions is the absolute path to the repository's agent
	// instructions, empty when it ships none.
	Instructions string
	// Onboarded reports whether the repository declares any agent context.
	Onboarded bool
	// Enabled is false when the repository opts out through its project
	// manifest.
	Enabled bool

	Skills []Skill
	Tools  []Tool
	Server *ServerSpec
	Issues []Issue
}

// Load reads every checkout in the workspace, in stable order.
//
// Checkouts that are not git working trees are skipped: the workspace reports
// any directory under its repos directory, but a scratch folder someone
// dropped there is not a repository and should not be treated as one.
func Load(ws *workspace.Workspace) ([]*Repo, error) {
	checkouts, err := ws.Checkouts()
	if err != nil {
		return nil, err
	}

	var repos []*Repo
	for _, c := range checkouts {
		if !c.IsGit {
			continue
		}
		repos = append(repos, LoadRepo(c.Dir))
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i].Name < repos[j].Name })
	return repos, nil
}

// LoadRepo reads one checkout. It never returns nil and never reports a hard
// error: anything wrong is recorded as an Issue so the rest of the workspace
// keeps working.
func LoadRepo(dir string) *Repo {
	name := filepath.Base(dir)
	r := &Repo{
		Name:      name,
		Dir:       dir,
		Namespace: SanitizeNamespace(name),
		Enabled:   true,
	}

	paths, issues := agentPaths(r)
	r.Issues = append(r.Issues, issues...)
	if !r.Enabled {
		return r
	}

	if instructions := filepath.Join(dir, filepath.FromSlash(paths.Instructions)); isFile(instructions) {
		r.Instructions = instructions
		r.Onboarded = true
	}

	skills, skillIssues := loadSkills(r, filepath.Join(dir, filepath.FromSlash(paths.Skills)))
	r.Skills = skills
	r.Issues = append(r.Issues, skillIssues...)
	if len(skills) > 0 {
		r.Onboarded = true
	}

	manifestPath := filepath.Join(dir, filepath.FromSlash(paths.MCP))
	if isFile(manifestPath) {
		r.Onboarded = true
		m, manifestIssues := loadManifest(r, manifestPath)
		r.Issues = append(r.Issues, manifestIssues...)
		if m != nil {
			if m.Namespace != "" {
				r.Namespace = m.Namespace
			}
			r.Description = m.Description
			r.Tools = m.Tools
			r.Server = m.Server
		}
	}
	return r
}

type agentPathSet struct {
	Instructions string
	Skills       string
	MCP          string
}

// agentPaths reads where a repository keeps its agent context, defaulting to
// the conventional locations. A repository following them needs no
// configuration at all.
func agentPaths(r *Repo) (agentPathSet, []Issue) {
	paths := agentPathSet{
		Instructions: DefaultInstructions,
		Skills:       DefaultSkills,
		MCP:          DefaultManifest,
	}

	path := filepath.Join(r.Dir, ".happy.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return paths, nil
	}

	var pm projectManifest
	if err := yaml.Unmarshal(data, &pm); err != nil {
		return paths, []Issue{{
			Repo:    r.Name,
			Path:    ".happy.yaml",
			Message: fmt.Sprintf("parsing agent section: %s", err.Error()),
		}}
	}

	if pm.Agent.Enabled != nil && !*pm.Agent.Enabled {
		r.Enabled = false
		return paths, nil
	}
	if pm.Agent.Instructions != "" {
		paths.Instructions = pm.Agent.Instructions
	}
	if pm.Agent.Skills != "" {
		paths.Skills = pm.Agent.Skills
	}
	if pm.Agent.MCP != "" {
		paths.MCP = pm.Agent.MCP
	}
	return paths, nil
}

// SanitizeNamespace makes a checkout directory name safe as a tool prefix.
//
// The default namespace comes from a directory name, which is not constrained
// the way an explicitly declared one is: happy-sdk.github.io would otherwise
// produce tool names like "happy-sdk.github.io__build", and clients commonly
// restrict tool names to [A-Za-z0-9_-].
func SanitizeNamespace(name string) string {
	var b strings.Builder
	var pendingDash bool
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteRune(r)
		default:
			pendingDash = true
		}
	}
	if b.Len() == 0 {
		return "repo"
	}
	return b.String()
}

// Usable reports whether the repository contributes anything.
func (r *Repo) Usable() bool {
	return r.Enabled && len(r.Issues) == 0 &&
		(len(r.Tools) > 0 || len(r.Skills) > 0 || r.Instructions != "" || r.Server != nil)
}

// QualifiedTool is the name a tool is exposed under. The prefix disambiguates
// an otherwise obvious name like "test" in a workspace holding several
// repositories.
func (r *Repo) QualifiedTool(tool string) string { return r.Namespace + "__" + tool }

// QualifiedSkill is the name a skill is exposed under.
func (r *Repo) QualifiedSkill(skill string) string { return r.Namespace + "/" + skill }

func isFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}
