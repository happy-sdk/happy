// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EntrypointStatus is what happened to one generated entrypoint.
type EntrypointStatus int

const (
	// EntrypointWritten means the file did not exist and was created.
	EntrypointWritten EntrypointStatus = iota
	// EntrypointKept means a file was already there and was left alone.
	EntrypointKept
)

func (s EntrypointStatus) String() string {
	if s == EntrypointWritten {
		return "written"
	}
	return "kept"
}

// EntrypointResult reports what EnsureEntrypoints did with one path.
type EntrypointResult struct {
	// Path is relative to the workspace root.
	Path   string
	Status EntrypointStatus
}

// InstructionsPath is the absolute path to the workspace's agent instructions,
// or empty when the marker declares none.
func (w *Workspace) InstructionsPath() string {
	if w.Config.Agents.Instructions == "" {
		return ""
	}
	return filepath.Join(w.Root, filepath.FromSlash(w.Config.Agents.Instructions))
}

// HasInstructions reports whether the declared instructions file is present.
// It can be absent legitimately: it usually lives inside a repository that has
// not been cloned yet.
func (w *Workspace) HasInstructions() bool {
	path := w.InstructionsPath()
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// EnsureEntrypoints materializes the configured entrypoint files at the
// workspace root, for agents that discover instructions by filename rather
// than by reading the marker.
//
// An existing file is never overwritten. The workspace is not version
// controlled, so anything already there belongs to whoever works here and
// there is no way to recover what a clobbering write would destroy.
func (w *Workspace) EnsureEntrypoints() ([]EntrypointResult, error) {
	var out []EntrypointResult
	for _, rel := range w.Config.Agents.entrypoints() {
		if err := validRelPath("agents.entrypoints", rel); err != nil {
			return out, err
		}
		path := filepath.Join(w.Root, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			out = append(out, EntrypointResult{Path: rel, Status: EntrypointKept})
			continue
		} else if !os.IsNotExist(err) {
			return out, fmt.Errorf("%w: %s: %s", Error, path, err.Error())
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return out, err
		}
		if err := os.WriteFile(path, []byte(w.EntrypointContent()), 0o600); err != nil {
			return out, err
		}
		out = append(out, EntrypointResult{Path: rel, Status: EntrypointWritten})
	}
	return out, nil
}

// EntrypointContent is the generated pointer written at the workspace root.
//
// It describes only the workspace's structure and where the real instructions
// are. It deliberately contains none of the organization's own conventions: a
// generated copy of guidance that lives elsewhere is a copy that goes stale,
// and this file is written once and never updated.
//
// It is also written to be useful when the instructions it points at are not
// there yet, since the repository holding them is usually cloned after the
// workspace is created.
func (w *Workspace) EntrypointContent() string {
	var b strings.Builder

	b.WriteString("# Workspace\n\n")
	b.WriteString("This directory is a development workspace, not a repository. ")
	b.WriteString("It is marked by `" + FileName + "`, which describes its layout.\n\n")

	fmt.Fprintf(&b, "- Repository checkouts live in `%s/`. Each is a separate repository with its own\n", w.Config.Layout.Repos)
	b.WriteString("  remote and review process; a change in one is a change to that project.\n")
	if scratch := w.Config.Layout.Scratch; scratch != "" {
		fmt.Fprintf(&b, "- `%s/` holds local notes and files belonging to whoever works here.\n", scratch)
	}
	b.WriteString("\n")

	if instructions := w.Config.Agents.Instructions; instructions != "" {
		fmt.Fprintf(&b, "## Instructions\n\nRead `%s`. It is the authority for this\n", instructions)
		b.WriteString("workspace and overrides anything in this file.\n\n")
		b.WriteString("If that file is missing, the repository holding it has not been cloned yet:\n\n")
		b.WriteString("```bash\nhappyctl workspace sync\n```\n\n")
	} else {
		b.WriteString("## Instructions\n\nThis workspace declares no instructions file. ")
		b.WriteString("Set `agents.instructions` in\n`" + FileName + "` to point at one.\n\n")
	}

	b.WriteString("## Per-repository context\n\n")
	b.WriteString("A repository may ship its own instructions, skills and tools in its `.happy/`\n")
	b.WriteString("directory. Those are specific to that repository and override anything more\n")
	b.WriteString("general. Check for them before working in one:\n\n")
	fmt.Fprintf(&b, "```bash\nls -a %s/<repo>/.happy/\n```\n\n", w.Config.Layout.Repos)

	b.WriteString("---\n\n")
	b.WriteString("<!--\n")
	b.WriteString("Generated once by `happyctl workspace init` and never updated.\n")
	b.WriteString("Edit it freely - it is yours, it is not tracked by any repository, and\n")
	b.WriteString("nothing will overwrite it.\n")
	b.WriteString("-->\n")

	return b.String()
}
