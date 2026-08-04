// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package discovery

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/goccy/go-yaml"
)

// Skill is a procedure a repository documents for a recurring task, read from
// .happy/skills/<name>/SKILL.md.
type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
	Version     string   `yaml:"version"`
	Requires    []string `yaml:"requires"`

	// Dir is the absolute path to the skill directory, which may hold scripts
	// or references beside SKILL.md.
	Dir string `yaml:"-"`
	// Path is the absolute path to SKILL.md.
	Path string `yaml:"-"`
	// Body is the Markdown after the frontmatter.
	Body string `yaml:"-"`
}

var frontmatterFence = []byte("---")

func loadSkills(r *Repo, dir string) ([]Skill, []Issue) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, []Issue{{Repo: r.Name, Path: relTo(r.Dir, dir), Message: err.Error()}}
	}

	var (
		skills []Skill
		issues []Issue
	)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, e.Name())
		path := filepath.Join(skillDir, "SKILL.md")
		if fi, err := os.Stat(path); err != nil || !fi.Mode().IsRegular() {
			issues = append(issues, Issue{
				Repo:    r.Name,
				Path:    relTo(r.Dir, skillDir),
				Message: "skill directory has no SKILL.md",
			})
			continue
		}
		s, err := parseSkill(path)
		if err != nil {
			issues = append(issues, Issue{Repo: r.Name, Path: relTo(r.Dir, path), Message: err.Error()})
			continue
		}
		// The directory name is the identity an agent sees; a mismatch means
		// one of the two is wrong and silently picking either hides it.
		if s.Name != e.Name() {
			issues = append(issues, Issue{
				Repo:    r.Name,
				Path:    relTo(r.Dir, path),
				Message: fmt.Sprintf("frontmatter name %q does not match directory %q", s.Name, e.Name()),
			})
			continue
		}
		s.Dir = skillDir
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, issues
}

// parseSkill reads a SKILL.md: a YAML frontmatter block fenced by --- lines,
// followed by the Markdown procedure.
func parseSkill(path string) (Skill, error) {
	var s Skill

	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}

	front, body, err := splitFrontmatter(data)
	if err != nil {
		return s, err
	}
	if err := yaml.Unmarshal(front, &s); err != nil {
		return s, fmt.Errorf("parsing frontmatter: %s", err.Error())
	}
	if s.Name == "" {
		return s, fmt.Errorf("frontmatter has no name")
	}
	if s.Description == "" {
		return s, fmt.Errorf("frontmatter has no description; it is what an agent matches against")
	}

	s.Path = path
	s.Body = string(bytes.TrimSpace(body))
	return s, nil
}

func splitFrontmatter(data []byte) (front, body []byte, err error) {
	trimmed := bytes.TrimLeft(data, "\ufeff \t\r\n")
	if !bytes.HasPrefix(trimmed, frontmatterFence) {
		return nil, nil, fmt.Errorf("missing YAML frontmatter: file must open with a --- fence")
	}
	rest := trimmed[len(frontmatterFence):]
	// Skip to the end of the opening fence line.
	if i := bytes.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[i+1:]
	} else {
		return nil, nil, fmt.Errorf("missing YAML frontmatter: unterminated opening fence")
	}

	end := bytes.Index(rest, append([]byte("\n"), frontmatterFence...))
	if end < 0 {
		return nil, nil, fmt.Errorf("missing YAML frontmatter: unterminated --- fence")
	}
	front = rest[:end]
	body = rest[end+1+len(frontmatterFence):]
	return front, body, nil
}

func relTo(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
