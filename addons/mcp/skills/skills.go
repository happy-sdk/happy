// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

// Package skills carries the skills built into the server.
//
// These exist to solve a bootstrapping problem: before any repository is
// cloned there is nothing to discover, so a workspace would have no way to
// explain itself. Built-in skills describe how a workspace is arranged and how
// to find the rest - structure, not any organization's conventions.
//
// They are served under their own namespace, so they can never shadow or be
// shadowed by a repository's skills - both are visible and it is always clear
// which is which. A repository's own skills are nonetheless the authority for
// work in it: they are reviewed, versioned and updatable, while anything
// embedded here can only change by releasing a new binary. Keep this set small
// for that reason, and confined to how a workspace is arranged rather than
// what any organization expects.
package skills

import (
	"embed"
	"io/fs"
)

//go:embed */SKILL.md
var files embed.FS

// Namespace is the prefix built-in skills are exposed under. It cannot collide
// with a repository namespace, which is derived from a checkout directory name
// and can never contain a dot.
const Namespace = "builtin"

// Skill is one built-in skill, kept deliberately close to what a repository
// ships so both can be served the same way.
type Skill struct {
	// Name is the directory name, which is also the skill's identity.
	Name string
	// Body is the full SKILL.md, frontmatter included.
	Body string
}

// All returns the built-in skills, in stable order.
func All() ([]Skill, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, err
	}

	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		body, err := fs.ReadFile(files, e.Name()+"/SKILL.md")
		if err != nil {
			return nil, err
		}
		out = append(out, Skill{Name: e.Name(), Body: string(body)})
	}
	return out, nil
}
