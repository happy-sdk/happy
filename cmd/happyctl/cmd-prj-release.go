// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package main

import (
	"fmt"
	"os/exec"

	"github.com/happy-sdk/addons/devel/project"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func (s *state) cmdProjectRelease() *command.Command {
	return command.New("release",
		command.Config{
			Description: "Release current project",
			Category:    "project",
		}).
		Disable(func(sess *session.Context) error {
			_, err := exec.LookPath("go")
			if err != nil {
				return err
			}
			prj, err := s.Project()
			if err != nil {
				return err
			}

			if !prj.Config().Get("releaser.enabled").Value().Bool() {
				return fmt.Errorf("%w: releaser disabled", project.Error)
			}
			return err
		}).
		WithFlags(
			cli.NewBoolFlag("skip-lint", false, "skip running linters"),
			cli.NewBoolFlag("skip-tests", false, "skip running tests"),
			cli.NewBoolFlag("skip-remote-checks", false, "skip running remote checks"),
			cli.NewBoolFlag("dirty", false, "allow dirty working directory"),
		).
		Before(func(sess *session.Context, args action.Args) error {
			prj, err := s.Project()
			if err != nil {
				return err
			}

			if prj.Config().Get("linter.enabled").Value().Bool() &&
				args.Flag("skip-lint").Var().Bool() {
				if err := prj.Config().Set("linter.enabled", false); err != nil {
					return err
				}
			}

			if prj.Config().Get("tests.enabled").Value().Bool() &&
				args.Flag("skip-tests").Var().Bool() {
				if err := prj.Config().Set("tests.enabled", false); err != nil {
					return err
				}
			}

			return nil
		}).
		Do(func(sess *session.Context, args action.Args) error {
			prj, err := s.Project()
			if err != nil {
				return err
			}
			return prj.Release(sess,
				args.Flag("dirty").Var().Bool(),
				args.Flag("skip-remote-checks").Var().Bool(),
			)
		})
}
