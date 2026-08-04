// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/happy-sdk/happy/addons/mcp/discovery"
	"github.com/happy-sdk/happy/addons/mcp/serve"
	"github.com/happy-sdk/happy/lib/workspace"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

// Command is the mcp command tree.
func Command() *command.Command {
	return command.New("mcp",
		command.Config{
			Category:    "workspace",
			Description: "Serve the workspace over the Model Context Protocol",
		}).
		AddInfo("Repositories declare their own tools and skills under .happy/, and this "+
			"serves whatever is found. Nothing about any repository is configured here, so "+
			"a repository that adds a tool is picked up without changing anything else.").
		WithSubCommands(
			cmdServe(),
			cmdList(),
			cmdDoctor(),
		)
}

func workspaceFlag() cli.Flag {
	return cli.NewStringFlag("workspace", "",
		"Workspace root; defaults to $HAPPY_WORKSPACE, then an upward search")
}

func cmdServe() *command.Command {
	return command.New("serve",
		command.Config{
			Description: "Serve over stdio; this is what MCP clients launch",
		}).
		WithFlags(workspaceFlag()).
		AddInfo("Launched by a client, not usually by hand. It never clones and never " +
			"writes to a repository: discovery reads what is already on disk.").
		Do(func(sess *session.Context, args action.Args) error {
			// stdout carries the JSON-RPC framing, so every byte of logging
			// has to go to stderr. Happy's console adapter writes to stdout by
			// default, and a single stray line corrupts the session in a way
			// that surfaces on the client as an unintelligible protocol error.
			if err := logging.ReplaceAdaptersStdout(sess.Log(), os.Stderr); err != nil {
				sess.Log().Debug("no stdout log adapter to redirect", "err", err.Error())
			}

			ws, err := resolveWorkspace(sess, args)
			if err != nil {
				return err
			}

			srv, err := serve.New(ws, serve.Options{
				Version: sess.Get("app.version").String(),
				Logf: func(format string, a ...any) {
					sess.Log().Debug(fmt.Sprintf(format, a...))
				},
			})
			if err != nil {
				return err
			}

			sess.Log().Notice("serving workspace",
				"root", ws.Root,
				"repositories", len(srv.Repos()))

			if err := srv.Run(sess.Context()); err != nil && !isDisconnect(err) {
				return err
			}
			sess.Log().Debug("client disconnected")
			return nil
		})
}

func cmdList() *command.Command {
	return command.New("list",
		command.Config{
			Description: "Show what would be served, and which repository each item comes from",
		}).
		WithFlags(workspaceFlag()).
		Do(func(sess *session.Context, args action.Args) error {
			ws, repos, err := load(sess, args)
			if err != nil {
				return err
			}

			fmt.Printf("workspace: %s\n\n", ws.Root)

			tbl := textfmt.NewTable(textfmt.TableWithHeader())
			tbl.AddRow("REPOSITORY", "NAMESPACE", "TOOLS", "SKILLS", "STATE")
			tbl.AddDivider()

			var tools, skillCount int
			for _, r := range repos {
				tools += len(r.Tools)
				skillCount += len(r.Skills)
				tbl.AddRow(r.Name, r.Namespace,
					fmt.Sprint(len(r.Tools)), fmt.Sprint(len(r.Skills)), repoState(r))
			}
			fmt.Println(tbl.String())

			for _, r := range repos {
				if len(r.Tools) == 0 && len(r.Skills) == 0 {
					continue
				}
				fmt.Printf("\n%s\n", r.Name)
				for _, t := range r.Tools {
					fmt.Printf("  tool   %s\n", r.QualifiedTool(t.Name))
				}
				for _, sk := range r.Skills {
					fmt.Printf("  skill  %s\n", r.QualifiedSkill(sk.Name))
				}
			}

			if tools+skillCount == 0 {
				fmt.Println("\nNo repository declares tools or skills yet. The built-in " +
					"skills are still served, so a client has something to work from.")
			}
			return nil
		})
}

func cmdDoctor() *command.Command {
	return command.New("doctor",
		command.Config{
			Description: "Validate every agent manifest and report why one was skipped",
		}).
		WithFlags(workspaceFlag()).
		AddInfo("A federated server fails invisibly: a repository whose manifest has a typo " +
			"silently contributes nothing, and the only symptom is a tool that is not there. " +
			"This names the file and the reason.").
		Do(func(sess *session.Context, args action.Args) error {
			ws, repos, err := load(sess, args)
			if err != nil {
				return err
			}

			fmt.Printf("workspace:    %s\n", ws.Root)
			fmt.Printf("repositories: %d\n\n", len(repos))

			var issues, onboarded int
			for _, r := range repos {
				if r.Onboarded {
					onboarded++
				}
				for _, is := range r.Issues {
					issues++
					fmt.Printf("  %s\n", is.String())
				}
			}

			if issues == 0 {
				fmt.Println("No manifest issues.")
			} else {
				fmt.Printf("\n%d issue(s) found.\n", issues)
			}
			fmt.Printf("%d of %d repositories declare agent context.\n", onboarded, len(repos))

			var optedOut []string
			for _, r := range repos {
				if !r.Enabled {
					optedOut = append(optedOut, r.Name)
				}
			}
			if len(optedOut) > 0 {
				fmt.Printf("Opted out via .happy.yaml: %s\n", strings.Join(optedOut, ", "))
			}

			if issues > 0 {
				// Non-zero so this is usable as a check, not only by eye.
				return fmt.Errorf("%w: %d manifest issue(s)", Error, issues)
			}
			return nil
		})
}

func load(sess *session.Context, args action.Args) (*workspace.Workspace, []*discovery.Repo, error) {
	ws, err := resolveWorkspace(sess, args)
	if err != nil {
		return nil, nil, err
	}
	repos, err := discovery.Load(ws)
	if err != nil {
		return nil, nil, err
	}
	return ws, repos, nil
}

// resolveWorkspace honours --workspace, then the environment, then ascends
// from the working directory. Clients launch servers with an unpredictable
// working directory, so the explicit forms matter more than the search does.
func resolveWorkspace(sess *session.Context, args action.Args) (*workspace.Workspace, error) {
	if flag := args.Flag("workspace"); flag.Present() {
		return workspace.Resolve(flag.String())
	}
	if wd := sess.Get("app.fs.path.wd").String(); wd != "" {
		if ws, err := workspace.Find(wd); err == nil {
			return ws, nil
		}
	}
	return workspace.Resolve("")
}

// repoState explains what a repository contributes, so an empty row is never
// ambiguous between "nothing declared" and "declared but broken".
func repoState(r *discovery.Repo) string {
	switch {
	case !r.Enabled:
		return "opted out"
	case !r.Onboarded:
		return "not onboarded"
	case len(r.Issues) > 0:
		return fmt.Sprintf("%d issue(s)", len(r.Issues))
	default:
		return "ok"
	}
}

// isDisconnect reports whether err is a client closing the connection, which
// is how a stdio session normally ends rather than a failure.
//
// The string match is deliberate and unfortunate. The SDK signals this with an
// internal sentinel wrapping the read error using %v rather than %w
// (internal/jsonrpc2/conn.go: `fmt.Errorf("%w: %v", errClosing, s.readErr)`),
// so the io.EOF is not in the chain for errors.Is to find, and the sentinel
// lives in an internal package that cannot be referenced from here. The
// errors.Is checks come first, so this is only reached when the chain carries
// nothing usable. Replace it if the SDK ever exports a sentinel.
func isDisconnect(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	return strings.Contains(err.Error(), "server is closing")
}
