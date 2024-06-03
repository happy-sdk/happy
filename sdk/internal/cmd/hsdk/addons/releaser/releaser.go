// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/addon"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/cli/command"
)

func Addon() *addon.Addon {
	r := newReleaser()

	addon := addon.New(addon.Config{
		Name: "Releaser",
	},
		addon.Option("wd", ".", "working directory of the project", false,
			func(key string, val vars.Value) error {
				str := val.String()
				if str == "." {
					return nil
				}
				if val.Empty() || str == "" || !filepath.IsAbs(str) {
					return fmt.Errorf("invalid value for %s: %q", key, val)
				}
				if wd, err := os.Stat(val.String()); err != nil {
					return fmt.Errorf("%s error: %w", key, err.Error())
				} else if !wd.IsDir() {
					return fmt.Errorf("%s is not a directory", val.String())
				}
				return nil
			}),

		addon.Option("next", "auto", "specify next version to release auto|major|minor|patch", false, nil),
		addon.Option("go.monorepo", false, "is project Go monorepo", false, nil),
		addon.Option("go.modules.count", 0, "total go modules found", false, nil),
		addon.Option("git.branch", "main", "git branch to release from", false,
			func(key string, val vars.Value) error {
				if val.Empty() {
					return fmt.Errorf("can not set empty branch for %s", key)
				}
				return nil
			}),
		addon.Option("git.remote.url", "-", "git remote url", false,
			func(key string, val vars.Value) error {
				if val.Empty() {
					return fmt.Errorf("can not set empty remote url for %s", key)
				}
				return nil
			}),
		addon.Option("git.remote.name", "origin", "git remote name", false,
			func(key string, val vars.Value) error {
				if val.Empty() {
					return fmt.Errorf("can not set empty remote url for %s", key)
				}
				return nil
			}),
		addon.Option("git.dirty", false, "set to true if there are uncommitted changes", false, nil),
		addon.Option("git.committer", "", "Name of the committer", false, nil),
		addon.Option("git.email", "", "Email of the committer", false, nil),
		addon.Option("git.allow.dirty", false, "Dirty git repo allowed", false, nil),
		addon.Option("github.token", "", "Github token for that repository with release permissions", false, nil),
	)

	addon.ProvideCommand(r.createReleaseCommand())

	return addon
}

func (r *releaser) createReleaseCommand() *command.Command {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := command.New(command.Config{
		Name:     "release",
		Usage:    "[path]",
		Category: "Maintanance",
		MaxArgs:  1,
	})

	cmd.AddInfo(`When no [path] argument is provided it searches for the application in the current directory.
  Optional [path] argument specifies application root directory.`)
	cmd.AddInfo(`
  EXAMPLES:
  hsdk release .
  hsdk release /path/to/app`)

	cmd.WithFlag(varflag.OptionFunc("next", []string{"auto"}, []string{"auto", "major", "minor", "patch"}, "specify next version to release", "n"))
	cmd.WithFlag(varflag.BoolFunc("dirty", false, "allow release from dirty git repository"))

	cmd.Before(func(sess *session.Context, args action.Args) error {
		path, err := args.ArgDefault(0, ".")
		if err != nil {
			return err
		}
		return r.Initialize(sess, path.String(), args.Flag("dirty").Present())
	})

	cmd.Do(func(sess *session.Context, args action.Args) error {
		return r.Run(args.Flag("next").String())
	})

	cmd.AfterAlways(func(sess *session.Context, err error) error {
		optstbl := textfmt.Table{}

		sess.Opts().WithPrefix("releaser.").Range(func(v vars.Variable) bool {
			optstbl.AddRow(v.Name(), v.Value().String())
			return true
		})
		sess.Log().Println(optstbl.String())
		return nil
	})

	return cmd
}
