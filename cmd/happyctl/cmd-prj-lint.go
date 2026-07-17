// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package main

import (
	"fmt"

	"github.com/happy-sdk/addons/devel/project"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func (s *state) cmdProjectLint() *command.Command {
	return command.New("lint",
		command.Config{
			Description: "Lint current project",
			Category:    "project",
		}).
		Disable(func(sess *session.Context) error {
			if !s.Found() {
				return fmt.Errorf("project not found")
			}
			prj, err := s.Project()
			if err != nil {
				return err
			}
			if !prj.Config().Get("linter.enabled").Value().Bool() {
				return fmt.Errorf("%w: linting disabled", project.Error)
			}
			return nil
		}).
		Do(func(sess *session.Context, args action.Args) error {
			prj, err := s.Project()
			if err != nil {
				return err
			}
			return prj.Lint(sess)
		})
}
