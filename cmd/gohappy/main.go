// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package main

import (
	"path/filepath"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects"
	"github.com/happy-sdk/happy/pkg/branding"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
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
		Slug:           "gohappy",
		Description:    "Happy Prototyping Framework and SDK",
		License:        "Apache-2.0",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2025,
		I18n: happy.I18nSettings{
			Language: "en",
		},
		CLI: happy.CliSettings{
			WithConfigCmd:        true,
			WithGlobalFlags:      true,
			HideDisabledCommands: false,
		},
		Logging: happy.LoggingSettings{
			WithSource: true,
			Level:      logging.LevelOk,
		},
	}).
		AddInfo("The Happy CLI is an experimental command-line tool designed to streamline management of Happy SDK-based projects. It simplifies project initialization, configuration, addon management, and release processes for single projects and monorepos. Additionally, it supports defining and running project-wide tasks to enhance development efficiency.").
		WithBrand(brand).
		WithAddon(
			projects.Addon(),
		).
		WithFlags(
			cli.NewStringFlag("wd", ".", "Working directory"),
		).BeforeAlways(func(sess *session.Context, args action.Args) error {

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

		return nil
	})

	app.Run()
}
