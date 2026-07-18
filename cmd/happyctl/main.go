// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package main

import (
	"path/filepath"
	"sync"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/addons/devel"
	"github.com/happy-sdk/happy/addons/devel/project"
	"github.com/happy-sdk/happy/addons/devel/projects"
	"github.com/happy-sdk/happy/pkg/branding"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/cmd/config"
	"github.com/happy-sdk/happy/sdk/session"
)

func main() {
	brand := branding.New(branding.Info{
		Name:    "Happy Theme",
		Slug:    "happy-theme",
		Version: "v1.0.0",
	})
	app := happy.New(&happy.Settings{
		Name:           "Happy SDK",
		Slug:           "happyctl",
		Description:    "Happy Prototyping Framework and SDK",
		License:        "Apache-2.0",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2025,
		Profiles: happy.ProfileSettings{
			AllowCustom: true,
		},
		I18n: happy.I18nSettings{
			Language: "en",
		},
		CLI: happy.CliSettings{
			WithGlobalFlags:      true,
			HideDisabledCommands: false,
			WithI18nCmd:          true,
		},
		Logging: happy.LoggingSettings{
			WithSource: true,
		},
		Instance: happy.InstanceSettings{},
	}).
		AddInfo("The Happy CLI is an experimental command-line tool designed to streamline management of Happy SDK-based projects. It simplifies project initialization, configuration, addon management, and release processes for single projects and monorepos. Additionally, it supports defining and running project-wide tasks to enhance development efficiency.").
		WithBrand(brand).
		WithAddon(
			devel.Addon(
				devel.Settings{
					Projects: projects.Settings{
						SearchPathIgnore: []string{
							"**/.Trash*",
							"**/vendor/*",
							"**/node_modules/*",
							"**/tmp/*",
							"**/.git/*",
						},
					},
				},
			),
		).
		WithFlags(
			cli.NewStringFlag("wd", ".", "Working directory"),
		)

	configCmdCnf := config.DefaultCommandConfig()
	configCmdCnf.DisableKeys = []string{
		"app.cli.hide_disabled_commands",
		"app.cli.main_max_args",
		"app.cli.main_min_args",
		"app.cli.with_config_cmd",
		"app.cli.with_global_flags",
		"app.devel.allow_prod",
		"app.i18n.supported",
		"app.instance.max",
		"app.logging.no_slog_default",
		"app.profiles.additional",
		"app.profiles.disabled",
		"app.profiles.enable_devel",
		"app.engine.throttle_ticks",
		"app.services.cron_on_service_start",
		// Options
		"app.firstrun",
		"app.is_devel",
		"app.main.exec.x",
	}

	s := &state{}

	app.WithCommands(
		cmdProjects(),
		config.Command(configCmdCnf),
		s.cmdProjectInfo(),
		s.cmdProjectLint(),
		s.cmdProjectRelease(),
		s.cmdProjectTest(),
	)

	app.BeforeAlways(func(sess *session.Context, args action.Args) error {

		if args.Flag("wd").Present() {
			var err error

			wd, err := filepath.Abs(args.Flag("wd").String())
			if err != nil {
				return err
			}

			if err := sess.Opts().Set("app.fs.path.wd", wd); err != nil {
				return err
			}
		}

		wd := sess.Get("app.fs.path.wd").String()
		dir, found, err := project.FindProjectDir(wd)
		if err != nil {
			return err
		}
		if found {
			s.markFound(dir)
			if err := s.open(sess); err != nil {
				return err
			}
		}
		return nil
	})

	app.Run()
}

type state struct {
	mu  sync.RWMutex
	prj *project.Project

	dir   string
	found bool
}

func (s *state) Project() (*project.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.prj == nil {
		return nil, project.ErrOpeningProject
	}
	return s.prj, nil
}

func (s *state) Found() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.found
}

func (s *state) markFound(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dir = dir
	s.found = true
}

func (s *state) open(sess *session.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.found {
		prj, err := project.Open(sess, s.dir)
		if err != nil {
			return err
		}
		s.prj = prj
		return nil
	}
	return project.ErrOpeningProject
}
