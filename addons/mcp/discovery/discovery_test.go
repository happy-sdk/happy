// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/lib/workspace"
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

// repoFixture builds a checkout inside a real workspace, so discovery is
// exercised through the same path the server uses.
func repoFixture(t *testing.T, name string, files map[string]string) (*workspace.Workspace, string) {
	t.Helper()

	root := t.TempDir()
	write(t, filepath.Join(root, workspace.FileName), "version: \"1\"\n")
	dir := filepath.Join(root, "src", name)
	write(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
	for rel, content := range files {
		write(t, filepath.Join(dir, filepath.FromSlash(rel)), content)
	}
	ws, err := workspace.Load(root)
	testutils.NoError(t, err)
	return ws, dir
}

const validManifest = `
version: "1"
namespace: demo
description: A demo repository.
tools:
  - name: build
    description: Build everything.
    exec:
      command: ["go", "build", "./..."]
      timeout: 90s
`

const validSkill = `---
name: do-a-thing
description: Do a thing. Use when a thing needs doing.
---

# Do a thing

Step one.
`

func TestLoadRepoConventionalPaths(t *testing.T) {
	_, dir := repoFixture(t, "demo", map[string]string{
		".happy/mcp.yaml":                   validManifest,
		".happy/AGENTS.md":                  "# demo\n",
		".happy/skills/do-a-thing/SKILL.md": validSkill,
	})

	r := LoadRepo(dir)
	testutils.Equal(t, 0, len(r.Issues), "unexpected issues")
	testutils.Assert(t, r.Onboarded, "expected onboarded")
	testutils.Equal(t, "demo", r.Namespace)
	testutils.Assert(t, r.Instructions != "", "expected AGENTS.md to be found")
	testutils.Equal(t, 1, len(r.Tools))
	testutils.Equal(t, 1, len(r.Skills))
	testutils.Equal(t, "demo__build", r.QualifiedTool("build"))
	testutils.Equal(t, "demo/do-a-thing", r.QualifiedSkill("do-a-thing"))
}

// A repository following the conventions needs no .happy.yaml at all; one that
// does not can move the paths.
func TestLoadRepoHonoursProjectManifestPaths(t *testing.T) {
	_, dir := repoFixture(t, "demo", map[string]string{
		".happy.yaml":                     "version: \"1\"\nagent:\n  instructions: docs/AGENTS.md\n  skills: docs/skills\n  mcp: docs/mcp.yaml\n",
		"docs/AGENTS.md":                  "# demo\n",
		"docs/mcp.yaml":                   validManifest,
		"docs/skills/do-a-thing/SKILL.md": validSkill,
		".happy/mcp.yaml":                 "version: \"9\"\n", // must be ignored
	})

	r := LoadRepo(dir)
	testutils.Equal(t, 0, len(r.Issues), "the conventional path must not be read when overridden")
	testutils.Equal(t, 1, len(r.Tools))
	testutils.Equal(t, 1, len(r.Skills))
	testutils.Assert(t, strings.HasSuffix(r.Instructions, filepath.Join("docs", "AGENTS.md")),
		"instructions = %q", r.Instructions)
}

// happyctl validates .happy.yaml strictly, but discovery only wants the agent
// section - a repository must not be punished here for configuration this
// package has no opinion about.
func TestLoadRepoIgnoresUnrelatedProjectConfig(t *testing.T) {
	_, dir := repoFixture(t, "demo", map[string]string{
		".happy.yaml":     "version: \"1\"\nreleaser:\n  enabled: true\nlinter:\n  golangci-lint:\n    disable: [staticcheck]\n",
		".happy/mcp.yaml": validManifest,
	})

	r := LoadRepo(dir)
	testutils.Equal(t, 0, len(r.Issues), "unrelated project config must not produce issues")
	testutils.Equal(t, 1, len(r.Tools))
}

func TestLoadRepoCanOptOut(t *testing.T) {
	_, dir := repoFixture(t, "demo", map[string]string{
		".happy.yaml":     "version: \"1\"\nagent:\n  enabled: false\n",
		".happy/mcp.yaml": validManifest,
	})

	r := LoadRepo(dir)
	testutils.Assert(t, !r.Enabled, "expected the repository to opt out")
	testutils.Equal(t, 0, len(r.Tools), "an opted-out repository must contribute nothing")
	testutils.Assert(t, !r.Usable(), "expected unusable")
}

func TestLoadRepoWithoutAgentContext(t *testing.T) {
	_, dir := repoFixture(t, "bare", nil)

	r := LoadRepo(dir)
	testutils.Assert(t, !r.Onboarded, "expected not onboarded")
	testutils.Equal(t, 0, len(r.Issues), "absence is not an error")
}

func TestManifestIssues(t *testing.T) {
	for _, tt := range []struct{ name, manifest, want string }{
		{"unsupported version", "version: \"2\"\n", "unsupported version"},
		{"malformed yaml", "version: \"1\"\ntools: [oops\n", "parsing manifest"},
		{
			"tool without description",
			"version: \"1\"\ntools:\n  - name: t\n    exec:\n      command: [\"go\"]\n",
			"no description",
		},
		{
			"duplicate tool",
			"version: \"1\"\ntools:\n  - name: t\n    description: d\n    exec:\n      command: [\"go\"]\n  - name: t\n    description: d\n    exec:\n      command: [\"go\"]\n",
			"duplicate tool",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, dir := repoFixture(t, "demo", map[string]string{".happy/mcp.yaml": tt.manifest})
			r := LoadRepo(dir)

			testutils.Assert(t, len(r.Issues) > 0, "expected an issue")
			var joined string
			for _, is := range r.Issues {
				joined += is.Message + "\n"
			}
			testutils.Assert(t, strings.Contains(joined, tt.want),
				"issues %q do not mention %q", joined, tt.want)
		})
	}
}

// One broken manifest must not empty the rest of the workspace.
func TestLoadIsolatesBrokenRepositories(t *testing.T) {
	ws, _ := repoFixture(t, "good", map[string]string{".happy/mcp.yaml": validManifest})
	bad := filepath.Join(ws.ReposDir(), "bad")
	write(t, filepath.Join(bad, ".git", "HEAD"), "ref: refs/heads/main\n")
	write(t, filepath.Join(bad, ".happy", "mcp.yaml"), "version: \"9\"\n")

	repos, err := Load(ws)
	testutils.NoError(t, err)
	testutils.Equal(t, 2, len(repos))

	byName := map[string]*Repo{}
	for _, r := range repos {
		byName[r.Name] = r
	}
	testutils.Equal(t, 1, len(byName["good"].Tools), "a neighbour's broken manifest affected this repository")
	testutils.Equal(t, 0, len(byName["good"].Issues))
	testutils.Assert(t, len(byName["bad"].Issues) > 0, "expected the broken repository to carry issues")
}

// The workspace reports every directory under its repos directory; only git
// working trees are repositories.
func TestLoadSkipsNonCheckouts(t *testing.T) {
	ws, _ := repoFixture(t, "real", nil)
	if err := os.MkdirAll(filepath.Join(ws.ReposDir(), "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	repos, err := Load(ws)
	testutils.NoError(t, err)
	testutils.Equal(t, 1, len(repos))
	testutils.Equal(t, "real", repos[0].Name)
}

func TestSanitizeNamespace(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"happy", "happy"},
		{"happy-sdk.github.io", "happy-sdk-github-io"},
		{"Mixed.Case", "mixed-case"},
		{"...", "repo"},
	} {
		testutils.Equal(t, tt.want, SanitizeNamespace(tt.in), "input %q", tt.in)
	}
}

func TestToolInputSchemaAlwaysObject(t *testing.T) {
	testutils.Equal(t, "object", (Tool{}).InputSchema()["type"])
}

// A placeholder naming a declared-but-omitted input renders empty; one naming
// something the schema never declares is a manifest bug.
func TestToolDeclaredInputs(t *testing.T) {
	tool := Tool{Input: map[string]any{
		"properties": map[string]any{"module": map[string]any{"type": "string"}},
	}}
	declared := tool.DeclaredInputs()
	testutils.Assert(t, declared["module"], "declared input not reported")
	testutils.Assert(t, !declared["nope"], "undeclared input must not be reported")
}
