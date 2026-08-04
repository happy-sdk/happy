// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// marked builds a workspace root with the given marker contents.
func marked(t *testing.T, marker string) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, FileName), marker)
	return root
}

// A marker carrying only a version must yield the recommended layout, so the
// simplest possible workspace needs no configuration at all.
func TestLoadAppliesDefaults(t *testing.T) {
	root := marked(t, "version: \"1\"\n")

	ws, err := Load(root)
	testutils.NoError(t, err)
	testutils.Equal(t, DefaultRepos, ws.Config.Layout.Repos)
	testutils.Equal(t, filepath.Join(root, "src"), ws.ReposDir())
}

// An omitted scratch directory defaults, but an explicitly empty one means the
// workspace has none - the two cannot be distinguished in YAML, so empty is
// treated as the deliberate choice.
func TestScratchCanBeDisabled(t *testing.T) {
	ws, err := Load(marked(t, "version: \"1\"\nlayout:\n  scratch: \"\"\n"))
	testutils.NoError(t, err)
	testutils.Equal(t, "", ws.ScratchDir(), "an empty scratch means the workspace has none")
}

func TestCustomLayout(t *testing.T) {
	root := marked(t, "version: \"1\"\nlayout:\n  repos: repositories\n  scratch: notes\n")

	ws, err := Load(root)
	testutils.NoError(t, err)
	testutils.Equal(t, filepath.Join(root, "repositories"), ws.ReposDir())
	testutils.Equal(t, filepath.Join(root, "notes"), ws.ScratchDir())
}

func TestLoadRejectsBadConfig(t *testing.T) {
	for _, tt := range []struct{ name, marker, want string }{
		{"unsupported version", "version: \"2\"\n", "unsupported version"},
		{"absolute repos", "version: \"1\"\nlayout:\n  repos: /tmp/x\n", "must be relative"},
		{"escaping repos", "version: \"1\"\nlayout:\n  repos: ../outside\n", "must stay inside"},
		{"repos is root", "version: \"1\"\nlayout:\n  repos: .\n", "must stay inside"},
		{
			"scratch collides with repos",
			"version: \"1\"\nlayout:\n  repos: src\n  scratch: src\n",
			"both",
		},
		{
			"org remote without placeholder",
			"version: \"1\"\norg:\n  remote: git@github.com:acme/happy.git\n",
			"must contain {repo}",
		},
		{
			"repo without name",
			"version: \"1\"\norg:\n  remote: git@x/{repo}.git\nrepos:\n  - dir: x\n",
			"has no name",
		},
		{
			"repo without any remote",
			"version: \"1\"\nrepos:\n  - name: happy\n",
			"no remote",
		},
		{
			"duplicate repo",
			"version: \"1\"\norg:\n  remote: git@x/{repo}.git\nrepos:\n  - name: a\n  - name: a\n",
			"duplicate repository",
		},
		{
			"colliding directories",
			"version: \"1\"\norg:\n  remote: git@x/{repo}.git\nrepos:\n  - name: a\n    dir: same\n  - name: b\n    dir: same\n",
			"both check out as",
		},
		{
			"nested repo directory",
			"version: \"1\"\norg:\n  remote: git@x/{repo}.git\nrepos:\n  - name: a\n    dir: nested/deep\n",
			"single path element",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(marked(t, tt.marker))
			testutils.Error(t, err)
			testutils.Assert(t, errors.Is(err, ErrConfig), "expected ErrConfig, got %v", err)
			testutils.Assert(t, strings.Contains(err.Error(), tt.want),
				"error %q does not mention %q", err.Error(), tt.want)
		})
	}
}

func TestFindAscends(t *testing.T) {
	root := marked(t, "version: \"1\"\n")
	deep := filepath.Join(root, "src", "repo", "pkg", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := Find(deep)
	testutils.NoError(t, err)
	testutils.Equal(t, root, ws.Root)
}

func TestFindReportsNotFound(t *testing.T) {
	_, err := Find(t.TempDir())
	testutils.Assert(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %v", err)
}

func TestResolveOrder(t *testing.T) {
	explicit := marked(t, "version: \"1\"\n")
	fromEnv := marked(t, "version: \"1\"\n")

	t.Run("explicit wins over environment", func(t *testing.T) {
		t.Setenv(EnvRoot, fromEnv)
		ws, err := Resolve(explicit)
		testutils.NoError(t, err)
		testutils.Equal(t, explicit, ws.Root)
	})

	t.Run("environment used when no explicit root", func(t *testing.T) {
		t.Setenv(EnvRoot, fromEnv)
		ws, err := Resolve("")
		testutils.NoError(t, err)
		testutils.Equal(t, fromEnv, ws.Root)
	})
}

func TestCreate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")

	ws, err := Create(root, Default())
	testutils.NoError(t, err)

	testutils.Assert(t, isDir(ws.ReposDir()), "repos directory not created")
	testutils.Assert(t, isDir(ws.ScratchDir()), "scratch directory not created")

	// The marker must round-trip: what Create writes, Load must accept.
	reloaded, err := Load(root)
	testutils.NoError(t, err)
	testutils.Equal(t, ws.Config.Layout.Repos, reloaded.Config.Layout.Repos)
	testutils.Equal(t, ws.Config.Layout.Scratch, reloaded.Config.Layout.Scratch)
}

func TestCreateSkipsScratchWhenDisabled(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ws")
	cnf := Default()
	cnf.Layout.Scratch = ""

	ws, err := Create(root, cnf)
	testutils.NoError(t, err)
	testutils.Equal(t, "", ws.ScratchDir())
	testutils.Assert(t, !isDir(filepath.Join(root, DefaultScratch)),
		"scratch directory must not be created when disabled")
}

// Overwriting a marker would silently discard whatever the workspace was
// configured to be.
func TestCreateRefusesToOverwrite(t *testing.T) {
	root := marked(t, "version: \"1\"\n")
	_, err := Create(root, Default())
	testutils.Assert(t, errors.Is(err, ErrExists), "expected ErrExists, got %v", err)
}

func TestCheckouts(t *testing.T) {
	root := marked(t, `
version: "1"
org:
  remote: git@github.com:acme/{repo}.git
repos:
  - name: declared
  - name: .github
    dir: org
  - name: absent
`)
	src := filepath.Join(root, "src")
	write(t, filepath.Join(src, "declared", ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(src, "org", ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(src, "undeclared", ".git", "HEAD"), "ref: refs/heads/main\n")
	if err := os.MkdirAll(filepath.Join(src, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := Load(root)
	testutils.NoError(t, err)

	checkouts, err := ws.Checkouts()
	testutils.NoError(t, err)
	testutils.Equal(t, 4, len(checkouts), "every directory is reported, not only declared checkouts")

	byName := map[string]Checkout{}
	for _, c := range checkouts {
		byName[c.Name] = c
	}

	testutils.Assert(t, byName["declared"].Declared(), "declared checkout not matched to its entry")
	testutils.Assert(t, byName["declared"].IsGit, "expected a git checkout")

	// The dir override is what keeps a repository named ".github" from
	// becoming a hidden directory.
	testutils.Assert(t, byName["org"].Declared(), "dir override not matched")
	testutils.Equal(t, ".github", byName["org"].Repo.Name)

	testutils.Assert(t, !byName["undeclared"].Declared(), "undeclared checkout must not match an entry")
	testutils.Assert(t, byName["undeclared"].IsGit, "expected a git checkout")

	// Present but not a checkout is a distinct state from absent.
	testutils.Assert(t, !byName["notes"].IsGit, "a plain directory is not a checkout")
}

func TestCheckoutsWithoutReposDir(t *testing.T) {
	ws, err := Load(marked(t, "version: \"1\"\n"))
	testutils.NoError(t, err)

	checkouts, err := ws.Checkouts()
	testutils.NoError(t, err, "an absent repos directory is empty, not an error")
	testutils.Equal(t, 0, len(checkouts))
}

func TestMissing(t *testing.T) {
	root := marked(t, `
version: "1"
org:
  remote: git@github.com:acme/{repo}.git
repos:
  - name: here
  - name: gone
`)
	write(t, filepath.Join(root, "src", "here", ".git", "HEAD"), "ref: refs/heads/main\n")

	ws, err := Load(root)
	testutils.NoError(t, err)

	missing, err := ws.Missing()
	testutils.NoError(t, err)
	testutils.Equal(t, 1, len(missing))
	testutils.Equal(t, "gone", missing[0].Name)
}

func TestRemoteFor(t *testing.T) {
	root := marked(t, `
version: "1"
org:
  remote: git@github.com:acme/{repo}.git
repos:
  - name: normal
  - name: elsewhere
    remote: git@gitlab.com:other/thing.git
`)
	ws, err := Load(root)
	testutils.NoError(t, err)

	got, err := ws.Config.RemoteFor("normal")
	testutils.NoError(t, err)
	testutils.Equal(t, "git@github.com:acme/normal.git", got)

	// A per-repository remote is what lets one workspace span hosts.
	got, err = ws.Config.RemoteFor("elsewhere")
	testutils.NoError(t, err)
	testutils.Equal(t, "git@gitlab.com:other/thing.git", got)

	// Undeclared repositories still resolve through the org template, so a
	// workspace need not list everything it can clone.
	got, err = ws.Config.RemoteFor("undeclared")
	testutils.NoError(t, err)
	testutils.Equal(t, "git@github.com:acme/undeclared.git", got)
}

func TestRemoteForWithoutOrgTemplate(t *testing.T) {
	ws, err := Load(marked(t, "version: \"1\"\n"))
	testutils.NoError(t, err)

	_, err = ws.Config.RemoteFor("anything")
	testutils.Assert(t, errors.Is(err, ErrConfig), "expected ErrConfig, got %v", err)
}

func TestDirHonoursOverride(t *testing.T) {
	root := marked(t, `
version: "1"
org:
  remote: git@github.com:acme/{repo}.git
repos:
  - name: .github
    dir: org
`)
	ws, err := Load(root)
	testutils.NoError(t, err)

	testutils.Equal(t, filepath.Join(root, "src", "org"), ws.Dir(".github"))
	testutils.Equal(t, filepath.Join(root, "src", "other"), ws.Dir("other"))
}

func isDir(path string) bool {
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
