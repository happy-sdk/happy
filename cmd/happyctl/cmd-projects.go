// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package main

import (
	"errors"
	"fmt"

	"github.com/happy-sdk/addons/devel"
	"github.com/happy-sdk/addons/devel/projects"
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func cmdProjects() *command.Command {
	return command.New("projects",
		command.Config{
			Category:    "Projects",
			Description: "Manage local projects known to Happy SDK",
		}).
		WithSubCommands(
			cmdProjectsLs(),
		)
}

func cmdProjectsLs() *command.Command {
	return command.New("ls",
		command.Config{
			Description: "List all projects",
		}).
		WithFlags(
			cli.NewBoolFlag("fresh", false, "Force project discovery, only when caching is enabled", "f"),
			cli.NewBoolFlag("all", false, "List all detectable projects including ones not directly depending on Happy SDK. E.g. git repositories", "a"),
			cli.NewBoolFlag("with-subprojects", false, "List also subprojects", "s"),
		).
		AddInfo("Use: (devel.projects.search_paths) to list paths or path patterns for project discovery").
		AddInfo("Use: (devel.projects.search_path_ignore) configuration option to define path patterns to ignore").
		Do(func(sess *session.Context, args action.Args) error {
			api, err := happy.API[*devel.API](sess)
			if err != nil {
				return err
			}
			prjs, err := api.Projects().List(sess,
				args.Flag("with-subprojects").Var().Bool(),
				args.Flag("all").Var().Bool(),
				args.Flag("fresh").Var().Bool(),
			)
			if err != nil {
				if errors.Is(err, projects.ErrNoProjectsFound) {
					sess.Log().Warn("no projects found")
					return nil
				}
				return err
			}

			table := textfmt.NewTable(
				textfmt.TableTitle("Projects"),
				textfmt.TableWithHeader(),
			)
			table.AddRow("Path", "Version", "Happy Version")
			for prj := range prjs {
				var ver string
				if prj.DependsOnHappy {
					ver = prj.HappyVersion.String()
				}
				table.AddRow(prj.Path, prj.Version.String(), ver)
			}
			fmt.Println(table.String())

			return nil
		})
}
