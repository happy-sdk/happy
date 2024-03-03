// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/internal/cmd/hap/addons/releaser/module"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk"
	"github.com/happy-sdk/happy/sdk/options"
)

type Settings struct {
	CommandEnabled settings.Bool `key:"command.enabled" default:"false"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func Addon(s Settings) *happy.Addon {
	addon := happy.NewAddon("releaser", s, opts()...)

	r := newReleaser()

	if s.CommandEnabled {
		addon.ProvidesCommand(r.createReleaseCommand())
	}

	return addon
}

func opts() []options.OptionSpec {
	return []options.OptionSpec{
		sdk.Option("working.directory", "/tmp/happy", "working directory of the project",
			func(key string, val vars.Value) error {
				if str := val.String(); val.Empty() || str == "" || !filepath.IsAbs(str) {
					return fmt.Errorf("invalid value for %s: %q", key, val)
				}
				return nil
			}),
		sdk.Option("next", "auto", "specify next version to release auto|major|minor|batch", nil),
		sdk.Option("go.monorepo", false, "is project Go monorepo", nil),
		sdk.Option("go.modules.count", 0, "total go modules found", nil),
		sdk.Option("git.branch", "main", "Git branch of the project",
			func(key string, val vars.Value) error {
				if val.Empty() {
					return fmt.Errorf("can not set empty branch for %s", key)
				}
				return nil
			}),
		sdk.Option("git.remote.url", "-", "URL of the remote repository",
			func(key string, val vars.Value) error {
				if val.Empty() {
					return fmt.Errorf("can not set empty remote url for %s", key)
				}
				return nil
			}),
		sdk.Option("git.remote.name", "origin", "Name of the remote repository",
			func(key string, val vars.Value) error {
				if val.Empty() {
					return fmt.Errorf("can not set empty remote url for %s", key)
				}
				return nil
			}),
		sdk.Option("git.dirty", false, "Set to true if there are uncommitted changes", nil),
		sdk.Option("git.committer", "", "Name of the committer", nil),
		sdk.Option("git.email", "", "Email of the committer", nil),
		sdk.Option("git.allow.dirty", false, "Dirty git repo allowed", nil),
		sdk.Option("github.token", "", "Github token for that repository with release permissions", nil),
	}
}

type releaser struct {
	happy.API
	mu       sync.RWMutex
	sess     *happy.Session
	config   configuration
	packages []*module.Package
	queue    []string
}

func newReleaser() *releaser {
	return &releaser{}
}

func (r *releaser) createReleaseCommand() *happy.Command {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := happy.NewCommand(
		"release",
		happy.Option("description", `Release a new version of the specific application.`),
		happy.Option("usage", "hap release [path]"),
		happy.Option("category", "devel"),
		happy.Option("argn.max", 1),
	)

	cmd.AddInfo(`When no [path] argument is provided it searches for the application in the current directory.
  Optional [path] argument specifies application root directory.`)
	cmd.AddInfo(`
  EXAMPLES:
  hap release .
  hap release /path/to/app`)

	cmd.AddFlag(varflag.OptionFunc("next", []string{"auto"}, []string{"auto", "major", "minor", "patch"}, "specify next version to release", "n"))
	cmd.AddFlag(varflag.BoolFunc("dirty", false, "allow release from dirty git repository"))

	cmd.Before(func(sess *happy.Session, args happy.Args) error {
		path, err := args.ArgDefault(0, ".")
		if err != nil {
			return err
		}

		if err := r.Initialize(sess, path.String(), args.Flag("dirty").Present()); err != nil {
			return err
		}

		return nil
	})

	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		return r.Run(args.Flag("next").String())
	})

	cmd.AfterSuccess(func(sess *happy.Session) error {
		return nil
	})

	cmd.AfterFailure(func(sess *happy.Session, err error) error {
		sess.Log().Error("release failed with error", slog.String("error", err.Error()))
		return nil
	})

	cmd.AfterAlways(func(sess *happy.Session, err error) error {
		return nil
	})

	return cmd
}
