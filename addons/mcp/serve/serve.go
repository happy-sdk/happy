// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

// Package serve exposes a discovered workspace over the Model Context
// Protocol.
//
// Tools are namespaced per repository, so a client configured once keeps
// working as repositories are added, removed, or grow new tools.
package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/happy-sdk/happy/addons/mcp/discovery"
	xexec "github.com/happy-sdk/happy/addons/mcp/exec"
	"github.com/happy-sdk/happy/addons/mcp/skills"
	"github.com/happy-sdk/happy/lib/workspace"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Namespace for the server's own tools, which describe the workspace rather
// than any one repository.
const Namespace = "workspace"

// Options configure a server.
type Options struct {
	Version string
	// Logf receives diagnostics. It must never write to stdout: under stdio
	// transport stdout carries the JSON-RPC framing and any stray byte
	// corrupts the session.
	Logf func(format string, args ...any)
}

// Server wraps an MCP server built from a workspace snapshot.
type Server struct {
	ws      *workspace.Workspace
	repos   []*discovery.Repo
	mcp     *mcp.Server
	logf    func(format string, args ...any)
	builtin []skills.Skill
}

// New builds a server from the workspace, registering the baseline tools and
// every declarative tool the checked-out repositories declare.
func New(ws *workspace.Workspace, opts Options) (*Server, error) {
	repos, err := discovery.Load(ws)
	if err != nil {
		return nil, err
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	if opts.Version == "" {
		opts.Version = "0.0.0-devel"
	}

	builtin, err := skills.All()
	if err != nil {
		return nil, err
	}

	s := &Server{
		ws:      ws,
		repos:   repos,
		logf:    opts.Logf,
		builtin: builtin,
		mcp: mcp.NewServer(&mcp.Implementation{
			Name:    "happy-sdk-mcp",
			Title:   "Happy SDK workspace",
			Version: opts.Version,
			Description: "Tools and skills discovered from the Happy SDK repositories " +
				"checked out in this workspace.",
		}, nil),
	}

	s.registerBaseline()
	s.registerRepos()
	return s, nil
}

// MCP exposes the underlying server, for transports other than Run.
func (s *Server) MCP() *mcp.Server { return s.mcp }

// Repos returns the workspace snapshot the server was built from.
func (s *Server) Repos() []*discovery.Repo { return s.repos }

// Run serves over stdio until the client disconnects or ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	return s.mcp.Run(ctx, &mcp.StdioTransport{})
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object"}
	if len(properties) > 0 {
		schema["properties"] = properties
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func textResult(s string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}}
}

func errorResult(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, args...)}},
	}
}

func (s *Server) registerBaseline() {
	s.mcp.AddTool(&mcp.Tool{
		Name:        Namespace + "__repos",
		Title:       "List workspace repositories",
		Description: "List the Happy SDK repositories checked out in this workspace, with their namespace and whether they declare agent context. Use this first to find out what is available to work on.",
		InputSchema: objectSchema(nil),
	}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var b strings.Builder
		fmt.Fprintf(&b, "workspace: %s\n\n", s.ws.Root)
		for _, r := range s.repos {
			fmt.Fprintf(&b, "%s\n  path:      src/%s\n  namespace: %s\n", r.Name, r.Name, r.Namespace)
			if r.Description != "" {
				fmt.Fprintf(&b, "  about:     %s\n", collapse(r.Description))
			}
			switch {
			case !r.Onboarded:
				b.WriteString("  agent:     not onboarded - no .happy directory\n")
			case len(r.Issues) > 0:
				fmt.Fprintf(&b, "  agent:     %d issue(s); run %s__doctor\n", len(r.Issues), Namespace)
			default:
				fmt.Fprintf(&b, "  agent:     %d tool(s), %d skill(s)%s\n",
					len(r.Tools), len(r.Skills), instructionsNote(r))
			}
			b.WriteString("\n")
		}
		return textResult(strings.TrimSpace(b.String())), nil
	})

	s.mcp.AddTool(&mcp.Tool{
		Name:        Namespace + "__skills",
		Title:       "List available skills",
		Description: "List every skill discovered across the workspace with its description. Skills are written procedures for recurring tasks. Call this to find one, then fetch its body with " + Namespace + "__skill.",
		InputSchema: objectSchema(map[string]any{
			"repo": map[string]any{
				"type":        "string",
				"description": "Restrict to one repository, by namespace.",
			},
		}),
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := decodeArgs(req)
		if err != nil {
			return errorResult("%s", err.Error()), nil
		}
		filter, _ := args["repo"].(string)

		var b strings.Builder
		var n int
		// Built-in skills first: they explain how a workspace is arranged and
		// are the only thing available before anything is cloned.
		if filter == "" || filter == skills.Namespace {
			for _, sk := range s.builtin {
				n++
				fmt.Fprintf(&b, "%s/%s\n  %s\n\n", skills.Namespace, sk.Name, collapse(builtinDescription(sk)))
			}
		}
		for _, r := range s.repos {
			if filter != "" && r.Namespace != filter {
				continue
			}
			for _, sk := range r.Skills {
				n++
				fmt.Fprintf(&b, "%s\n  %s\n", r.QualifiedSkill(sk.Name), collapse(sk.Description))
				if len(sk.Keywords) > 0 {
					fmt.Fprintf(&b, "  keywords: %s\n", strings.Join(sk.Keywords, ", "))
				}
				b.WriteString("\n")
			}
		}
		if n == 0 {
			return textResult("No skills found. Repositories declare them under .happy/skills/<name>/SKILL.md."), nil
		}
		return textResult(strings.TrimSpace(b.String())), nil
	})

	s.mcp.AddTool(&mcp.Tool{
		Name:        Namespace + "__skill",
		Title:       "Read a skill",
		Description: "Fetch the full procedure for one skill, by its qualified name such as happy/release-a-module. Bodies are fetched on demand rather than pushed into context, so list first and read only what you need.",
		InputSchema: objectSchema(map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Qualified skill name, <namespace>/<skill>.",
			},
		}, "name"),
	}, func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := decodeArgs(req)
		if err != nil {
			return errorResult("%s", err.Error()), nil
		}
		name, _ := args["name"].(string)
		ns, skill, ok := strings.Cut(name, "/")
		if !ok {
			return errorResult("name must be <namespace>/<skill>, got %q", name), nil
		}
		if ns == skills.Namespace {
			for _, sk := range s.builtin {
				if sk.Name == skill {
					return textResult(sk.Body), nil
				}
			}
			return errorResult("no built-in skill %q", skill), nil
		}
		for _, r := range s.repos {
			if r.Namespace != ns {
				continue
			}
			for _, sk := range r.Skills {
				if sk.Name != skill {
					continue
				}
				var b strings.Builder
				fmt.Fprintf(&b, "# %s\n\nrepository: src/%s\nfile: %s\n\n---\n\n%s\n",
					r.QualifiedSkill(sk.Name), r.Name, sk.Path, sk.Body)
				return textResult(b.String()), nil
			}
		}
		return errorResult("no skill %q; call %s__skills to list what exists", name, Namespace), nil
	})

	s.mcp.AddTool(&mcp.Tool{
		Name:        Namespace + "__doctor",
		Title:       "Validate agent manifests",
		Description: "Report why a repository contributes no tools or skills. A federated server fails invisibly - a typo in one manifest silently removes that repository - so check here when something you expect is missing.",
		InputSchema: objectSchema(nil),
	}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var b strings.Builder
		var issues int
		for _, r := range s.repos {
			for _, is := range r.Issues {
				issues++
				fmt.Fprintf(&b, "%s\n", is.String())
			}
		}
		if issues == 0 {
			fmt.Fprintf(&b, "No manifest issues across %d repositories.\n", len(s.repos))
		}
		var notOnboarded []string
		for _, r := range s.repos {
			if !r.Onboarded {
				notOnboarded = append(notOnboarded, r.Name)
			}
		}
		if len(notOnboarded) > 0 {
			fmt.Fprintf(&b, "\nNot onboarded (no .happy directory): %s\n", strings.Join(notOnboarded, ", "))
		}
		return textResult(strings.TrimSpace(b.String())), nil
	})
}

func (s *Server) registerRepos() {
	for _, r := range s.repos {
		if len(r.Issues) > 0 {
			s.logf("skipping %s: %d manifest issue(s)", r.Name, len(r.Issues))
			continue
		}
		for _, t := range r.Tools {
			s.registerTool(r, t)
		}
		if r.Server != nil {
			// Delegated servers are declared but not yet spawned. Say so
			// rather than silently ignoring the declaration.
			s.logf("%s declares a downstream server; proxying is not implemented yet", r.Name)
		}
	}
}

func (s *Server) registerTool(r *discovery.Repo, t discovery.Tool) {
	name := r.QualifiedTool(t.Name)
	tool := &mcp.Tool{
		Name:        name,
		Title:       t.Title,
		Description: strings.TrimSpace(t.Description),
		InputSchema: t.InputSchema(),
	}

	s.mcp.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := decodeArgs(req)
		if err != nil {
			return errorResult("%s", err.Error()), nil
		}

		res, err := xexec.Run(ctx, xexec.Request{
			RepoDir:  r.Dir,
			Command:  t.Exec.Command,
			Cwd:      t.Exec.Cwd,
			Args:     toArgRules(t.Exec.Args),
			Inputs:   args,
			Declared: t.DeclaredInputs(),
			Timeout:  t.Exec.Timeout.Duration(xexec.DefaultTimeout),
		})
		if err != nil {
			if res != nil && res.Output != "" {
				return errorResult("%s\n\n%s", err.Error(), res.Output), nil
			}
			return errorResult("%s", err.Error()), nil
		}

		out := strings.TrimRight(res.Output, "\n")
		if out == "" {
			out = fmt.Sprintf("(no output, exit %d)", res.ExitCode)
		}
		if res.Truncated {
			out += fmt.Sprintf("\n\n[output truncated at %d bytes]", xexec.MaxOutput)
		}
		if res.ExitCode != 0 {
			// A non-zero exit is the answer to questions like "do the tests
			// pass", so return the output as a tool error rather than
			// discarding it.
			return errorResult("exit %d after %s\n\n%s", res.ExitCode, res.Duration.Round(time.Millisecond), out), nil
		}
		return textResult(out), nil
	})
	s.logf("registered %s", name)
}

func toArgRules(in []discovery.ArgRule) []xexec.ArgRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]xexec.ArgRule, 0, len(in))
	for _, r := range in {
		out = append(out, xexec.ArgRule{If: r.If, Add: r.Add})
	}
	return out
}

// decodeArgs unmarshals the raw arguments. AddTool's low-level handler does no
// validation, which is what dynamically declared schemas need, so callers
// check what they use.
func decodeArgs(req *mcp.CallToolRequest) (map[string]any, error) {
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("cannot decode arguments: %s", err.Error())
	}
	if args == nil {
		args = map[string]any{}
	}
	return args, nil
}

func instructionsNote(r *discovery.Repo) string {
	if r.Instructions == "" {
		return ""
	}
	return ", AGENTS.md"
}

func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// builtinDescription pulls the description out of a built-in skill's
// frontmatter for listing. The body is served verbatim, so the frontmatter is
// parsed only for this summary and a malformed one costs a description rather
// than the skill.
func builtinDescription(sk skills.Skill) string {
	const key = "description:"
	for line := range strings.SplitSeq(sk.Body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, key) {
			continue
		}
		if desc := strings.TrimSpace(strings.TrimPrefix(trimmed, key)); desc != "" && desc != ">" {
			return desc
		}
		// A folded block scalar puts the text on the following lines; the
		// first of them is enough for a listing.
		rest := strings.SplitN(sk.Body, key, 2)[1]
		for l := range strings.SplitSeq(rest, "\n") {
			if t := strings.TrimSpace(l); t != "" && t != ">" {
				return t
			}
		}
	}
	return "(no description)"
}
