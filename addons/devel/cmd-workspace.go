// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2026 The Happy Authors

package devel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/addons/devel/pkg/views"
	"github.com/happy-sdk/happy/lib/scm/gitutils"
	"github.com/happy-sdk/happy/lib/workspace"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"golang.org/x/term"
)

// workspaceCommand is the command tree for creating and maintaining a
// workspace: a directory holding several checkouts side by side.
func workspaceCommand() *command.Command {
	return command.New("workspace",
		command.Config{
			Category:    "workspace",
			Description: "Create and maintain a workspace of repository checkouts",
		}).
		AddInfo("A workspace is local state, not a repository: a marked directory holding "+
			"checkouts side by side plus your own scratch space. Repositories are kept in "+
			"separate checkouts rather than nested inside another repository, which breaks "+
			"tools that resolve a project root by ascending.").
		AddInfo("Nothing here is specific to one organization. The marker names where "+
			"repositories come from, so the same commands serve any set of repositories "+
			"developed together.").
		WithSubCommands(
			cmdWorkspaceInit(),
			cmdWorkspaceInfo(),
			cmdWorkspaceClone(),
			cmdWorkspaceSync(),
		)
}

func cmdWorkspaceInit() *command.Command {
	return command.New("init",
		command.Config{
			Description: "Create a workspace in the current or given directory",
			MaxArgs:     1,
		}).
		AddInfo("Writes the workspace marker, creates its directories, and generates the "+
			"agent entrypoints. Existing files are never overwritten.").
		WithFlags(
			cli.NewStringFlag("org", "", "Organization name"),
			cli.NewStringFlag("remote", "", "Clone URL template, e.g. git@github.com:acme/{repo}.git"),
			cli.NewStringFlag("repos", "", "Comma-separated repositories to declare"),
			cli.NewStringFlag("repos-dir", workspace.DefaultRepos, "Directory holding checkouts"),
			cli.NewStringFlag("scratch", workspace.DefaultScratch, "Directory for your own files"),
			cli.NewBoolFlag("no-scratch", false, "Do not create a scratch directory"),
			cli.NewStringFlag("instructions", "", "Path to the workspace agent instructions"),
			cli.NewBoolFlag("no-entrypoint", false, "Do not generate agent entrypoint files"),
			cli.NewBoolFlag("clone", false, "Clone the declared repositories"),
			cli.NewBoolFlag("yes", false, "Do not prompt; use flags and defaults", "y"),
		).
		Do(func(sess *session.Context, args action.Args) error {
			root, err := initRoot(sess, args)
			if err != nil {
				return err
			}

			answers := views.WorkspaceAnswers{
				Root:         root,
				Org:          args.Flag("org").String(),
				Remote:       args.Flag("remote").String(),
				Repos:        args.Flag("repos").String(),
				ReposDir:     args.Flag("repos-dir").String(),
				Scratch:      args.Flag("scratch").String(),
				Instructions: args.Flag("instructions").String(),
				Clone:        args.Flag("clone").Var().Bool(),
			}
			if args.Flag("no-scratch").Var().Bool() {
				answers.Scratch = ""
			}

			// The wizard only fills in the flags, so everything it can do is
			// also scriptable. It is skipped without a terminal, which is how
			// this runs in CI and from other tools.
			if !args.Flag("yes").Var().Bool() && interactive() {
				wizard, err := runWorkspaceWizard(answers)
				if err != nil {
					return err
				}
				if wizard.Cancelled {
					fmt.Println("Cancelled; nothing was created.")
					return nil
				}
				answers = wizard.Answers
				root = answers.Root
			}

			cnf := workspace.Default()
			cnf.Org.Name = answers.Org
			cnf.Org.Remote = answers.Remote
			cnf.Layout.Repos = answers.ReposDir
			cnf.Layout.Scratch = answers.Scratch
			cnf.Agents.Instructions = answers.Instructions
			if args.Flag("no-entrypoint").Var().Bool() {
				// Explicitly empty, not omitted: omitted means "use the
				// default", which is the opposite of what was asked for.
				cnf.Agents.Entrypoints = []string{}
			}

			for _, name := range splitList(answers.Repos) {
				cnf.Repos = append(cnf.Repos, workspace.Repo{Name: name})
			}

			ws, err := workspace.Create(root, cnf)
			if err != nil {
				return err
			}

			fmt.Printf("Created workspace at %s\n", ws.Root)
			fmt.Printf("  marker    %s\n", workspace.FileName)
			fmt.Printf("  repos     %s/\n", cnf.Layout.Repos)
			if cnf.Layout.Scratch != "" {
				fmt.Printf("  scratch   %s/\n", cnf.Layout.Scratch)
			}

			entrypoints, err := ws.EnsureEntrypoints()
			if err != nil {
				return err
			}
			for _, e := range entrypoints {
				fmt.Printf("  %-9s %s\n", e.Status, e.Path)
			}

			if answers.Clone {
				if err := cloneMissing(sess, ws); err != nil {
					return err
				}
			} else if len(cnf.Repos) > 0 {
				fmt.Printf("\n%d repository declaration(s) written. Clone them with:\n", len(cnf.Repos))
				fmt.Printf("  happyctl workspace sync\n")
			}
			return nil
		})
}

func cmdWorkspaceInfo() *command.Command {
	return command.New("info",
		command.Config{
			Description: "Show the workspace layout and what is checked out",
		}).
		WithFlags(
			cli.NewStringFlag("workspace", "", "Workspace root; defaults to $HAPPY_WORKSPACE, then an upward search"),
		).
		Do(func(sess *session.Context, args action.Args) error {
			ws, err := resolveWorkspace(sess, args)
			if err != nil {
				return err
			}

			fmt.Printf("workspace: %s\n", ws.Root)
			if ws.Config.Org.Name != "" {
				fmt.Printf("org:       %s\n", ws.Config.Org.Name)
			}
			fmt.Printf("repos:     %s/\n", ws.Config.Layout.Repos)
			if scratch := ws.Config.Layout.Scratch; scratch != "" {
				fmt.Printf("scratch:   %s/\n", scratch)
			} else {
				fmt.Printf("scratch:   (none)\n")
			}
			if instr := ws.Config.Agents.Instructions; instr != "" {
				state := "missing - clone the repository that holds it"
				if ws.HasInstructions() {
					state = "present"
				}
				fmt.Printf("agents:    %s (%s)\n", instr, state)
			}
			fmt.Println()

			checkouts, err := ws.Checkouts()
			if err != nil {
				return err
			}
			missing, err := ws.Missing()
			if err != nil {
				return err
			}

			if len(checkouts) == 0 && len(missing) == 0 {
				fmt.Println("Nothing checked out yet.")
				return nil
			}

			tbl := textfmt.NewTable(textfmt.TableWithHeader())
			tbl.AddRow("DIRECTORY", "STATE", "DECLARED")
			tbl.AddDivider()
			for _, c := range checkouts {
				state := "not a checkout"
				if c.IsGit {
					state = "ok"
				}
				tbl.AddRow(c.Name, state, fmt.Sprint(c.Declared()))
			}
			for _, m := range missing {
				tbl.AddRow(m.LocalDir(), "missing", "true")
			}
			fmt.Println(tbl.String())

			if len(missing) > 0 {
				fmt.Printf("\n%d declared repository(ies) not checked out. Run:\n", len(missing))
				fmt.Printf("  happyctl workspace sync\n")
			}
			return nil
		})
}

func cmdWorkspaceClone() *command.Command {
	return command.New("clone",
		command.Config{
			Description: "Clone a repository into the workspace and declare it",
			MinArgs:     1,
			MinArgsErr:  "no repository name given",
			MaxArgs:     1,
		}).
		WithFlags(
			cli.NewStringFlag("workspace", "", "Workspace root; defaults to $HAPPY_WORKSPACE, then an upward search"),
		).
		AddInfo("The name is resolved against the workspace's remote template, unless the " +
			"marker gives the repository a remote of its own. Cloning also records the " +
			"declaration, so the marker grows as you work rather than needing to be " +
			"complete up front.").
		Do(func(sess *session.Context, args action.Args) error {
			ws, err := resolveWorkspace(sess, args)
			if err != nil {
				return err
			}
			name, err := args.ArgDefault(0, "")
			if err != nil {
				return err
			}
			return cloneRepo(sess, ws, workspace.Repo{Name: name.String()}, true)
		})
}

func cmdWorkspaceSync() *command.Command {
	return command.New("sync",
		command.Config{
			Description: "Clone declared repositories that are not checked out",
		}).
		WithFlags(
			cli.NewStringFlag("workspace", "", "Workspace root; defaults to $HAPPY_WORKSPACE, then an upward search"),
		).
		Do(func(sess *session.Context, args action.Args) error {
			ws, err := resolveWorkspace(sess, args)
			if err != nil {
				return err
			}
			return cloneMissing(sess, ws)
		})
}

// resolveWorkspace honours --workspace, then the environment, then ascends
// from the working directory. happyctl's --wd is respected so a workspace can
// be operated on from elsewhere.
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

func initRoot(sess *session.Context, args action.Args) (string, error) {
	arg, err := args.ArgDefault(0, "")
	if err != nil {
		return "", err
	}
	if root := arg.String(); root != "" {
		return filepath.Abs(root)
	}
	if wd := sess.Get("app.fs.path.wd").String(); wd != "" {
		return wd, nil
	}
	return os.Getwd()
}

// cloneMissing clones every declared repository that is not checked out.
func cloneMissing(sess *session.Context, ws *workspace.Workspace) error {
	missing, err := ws.Missing()
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		fmt.Println("Nothing to clone: every declared repository is checked out.")
		return nil
	}
	for _, repo := range missing {
		if err := cloneRepo(sess, ws, repo, false); err != nil {
			return err
		}
	}
	return nil
}

// cloneRepo clones one repository, optionally recording it in the marker.
func cloneRepo(sess *session.Context, ws *workspace.Workspace, repo workspace.Repo, declare bool) error {
	remote, err := ws.Config.RemoteFor(repo.Name)
	if err != nil {
		return err
	}
	dir := ws.Dir(repo.Name)
	if _, err := os.Stat(dir); err == nil {
		fmt.Printf("%-24s already checked out\n", repo.Name)
		return nil
	}

	fmt.Printf("%-24s cloning from %s\n", repo.Name, remote)
	if err := gitutils.Clone(sess, remote, dir); err != nil {
		return err
	}

	if declare {
		if _, already := ws.Config.Repo(repo.Name); !already {
			ws.Config.Repos = append(ws.Config.Repos, repo)
			if err := ws.Save(); err != nil {
				return err
			}
			fmt.Printf("%-24s declared in %s\n", repo.Name, workspace.FileName)
		}
	}
	return nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// interactive reports whether there is a terminal to prompt on. Without one -
// in CI, or when another tool drives happyctl - the wizard must not run, since
// it would block forever on input that is never coming.
func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func runWorkspaceWizard(seed views.WorkspaceAnswers) (views.WorkspaceWizard, error) {
	model, err := tea.NewProgram(views.NewWorkspaceWizard(seed)).Run()
	if err != nil {
		return views.WorkspaceWizard{}, err
	}
	wizard, ok := model.(views.WorkspaceWizard)
	if !ok {
		return views.WorkspaceWizard{}, fmt.Errorf("%w: unexpected wizard result", Error)
	}
	return wizard, nil
}
