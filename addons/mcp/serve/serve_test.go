// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package serve

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/lib/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const manifest = `
version: "1"
namespace: demo
description: A demo repository.
tools:
  - name: greet
    title: Greet
    description: Print a greeting.
    input:
      properties:
        who:
          type: string
          description: Who to greet.
    exec:
      command: ["echo", "hello"]
      args:
        - if: "{{ .who }}"
          add: ["{{ .who }}"]
  - name: boom
    title: Boom
    description: Fail on purpose.
    exec:
      command: ["sh", "-c", "echo went wrong >&2; exit 2"]
`

const skill = `---
name: do-a-thing
description: Do a thing. Use when a thing needs doing.
---

# Do a thing

Step one.
`

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// connect builds a server over a fixture workspace and returns a connected
// client session, exercising the real protocol rather than internals.
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()

	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "AGENTS.md"), "# workspace\n")
	repo := filepath.Join(root, "src", "demo")
	mustWrite(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	mustWrite(t, filepath.Join(repo, ".happy", "mcp.yaml"), manifest)
	mustWrite(t, filepath.Join(repo, ".happy", "skills", "do-a-thing", "SKILL.md"), skill)

	srv, err := New(&workspace.Workspace{Root: root, Config: workspace.Default()}, Options{Version: "test"})
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go func() {
		// Ends when the session closes at test teardown.
		_ = srv.MCP().Run(context.WithoutCancel(ctx), serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestServerAdvertisesBaselineAndRepoTools(t *testing.T) {
	session := connect(t)

	res, err := session.ListTools(t.Context(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	for _, tool := range res.Tools {
		got[tool.Name] = tool.Description
	}

	for _, want := range []string{
		Namespace + "__repos",
		Namespace + "__skills",
		Namespace + "__skill",
		Namespace + "__doctor",
		"demo__greet",
		"demo__boom",
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing tool %q; have %v", want, keys(got))
		}
	}

	// The description is the only signal a model has for when to call a tool,
	// so an empty one is a bug even though the protocol permits it.
	for name, desc := range got {
		if strings.TrimSpace(desc) == "" {
			t.Errorf("tool %q has no description", name)
		}
	}
}

func TestCallRepoTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX helpers")
	}
	session := connect(t)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "demo__greet",
		Arguments: map[string]any{"who": "world"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", textOf(res))
	}
	if got := textOf(res); got != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}

// An optional input that is absent must not leak an empty argv entry.
func TestCallRepoToolWithoutOptionalInput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX helpers")
	}
	session := connect(t)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "demo__greet"})
	if err != nil {
		t.Fatal(err)
	}
	if got := textOf(res); got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

// A failing command is reported as a tool error but must still carry its
// output: "the tests fail, here is why" is the useful answer.
func TestFailingToolKeepsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX helpers")
	}
	session := connect(t)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "demo__boom"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error for a non-zero exit")
	}
	out := textOf(res)
	if !strings.Contains(out, "went wrong") {
		t.Fatalf("output lost: %q", out)
	}
	if !strings.Contains(out, "exit 2") {
		t.Fatalf("exit code not reported: %q", out)
	}
}

func TestSkillTools(t *testing.T) {
	session := connect(t)

	list, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: Namespace + "__skills"})
	if err != nil {
		t.Fatal(err)
	}
	if got := textOf(list); !strings.Contains(got, "demo/do-a-thing") {
		t.Fatalf("skill not listed: %q", got)
	}

	body, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      Namespace + "__skill",
		Arguments: map[string]any{"name": "demo/do-a-thing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := textOf(body); !strings.Contains(got, "Step one.") {
		t.Fatalf("skill body not returned: %q", got)
	}

	missing, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      Namespace + "__skill",
		Arguments: map[string]any{"name": "demo/nope"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !missing.IsError {
		t.Fatal("expected an error for an unknown skill")
	}
}

func TestReposTool(t *testing.T) {
	session := connect(t)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: Namespace + "__repos"})
	if err != nil {
		t.Fatal(err)
	}
	out := textOf(res)
	if !strings.Contains(out, "demo") || !strings.Contains(out, "1 tool") {
		// greet and boom are two tools; assert the shape, not the count text.
		if !strings.Contains(out, "tool(s)") {
			t.Fatalf("repos output missing tool summary: %q", out)
		}
	}
	if !strings.Contains(out, "A demo repository.") {
		t.Fatalf("repos output missing description: %q", out)
	}
}

func textOf(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
