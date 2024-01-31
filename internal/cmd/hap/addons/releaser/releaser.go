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
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/settings"
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
	addon := happy.NewAddon("releaser", s)

	r := newReleaser()

	if s.CommandEnabled {
		addon.ProvidesCommand(r.createReleaseCommand())
	}

	addon.Option("working.directory", "/tmp/happy", "working directory of the project", func(key string, val vars.Value) error {
		if str := val.String(); val.Empty() || str == "" || !filepath.IsAbs(str) {
			return fmt.Errorf("invalid value for %s: %q", key, val)
		}
		return nil
	})

	addon.Option("next", "auto", "specify next version to release auto|major|minor|batch", nil)
	addon.Option("go.monorepo", false, "is project Go monorepo", nil)
	addon.Option("go.modules.count", 0, "total go modules found", nil)
	addon.Option("git.branch", "main", "branch to release from", nil)
	addon.Option("git.remote.url", "", "URL of the remote repository", nil)
	addon.Option("git.remote.name", "", "Name of the remote repository", nil)
	addon.Option("git.dirty", "", "true if there are uncommitted changes", nil)
	addon.Option("git.committer", "", "name of the committer", nil)
	addon.Option("git.email", "", "email of the committer", nil)
	addon.Option("github.token", "", "committer github token for that repository with release permissions", nil)

	return addon
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

	cmd.Before(func(sess *happy.Session, args happy.Args) error {
		path, err := args.ArgDefault(0, ".")
		if err != nil {
			return err
		}
		if err := r.Initialize(sess, path.String()); err != nil {
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
