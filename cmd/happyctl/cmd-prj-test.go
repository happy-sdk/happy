// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package main

import (
	"fmt"
	"os/exec"

	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func (s *state) cmdProjectTest() *command.Command {
	return command.New("test",
		command.Config{
			Description: "Run project tests",
			Category:    "project",
		}).
		Disable(func(sess *session.Context) error {
			if !s.Found() {
				return fmt.Errorf("project not found")
			}
			_, err := exec.LookPath("go")
			return err
		}).
		Do(func(sess *session.Context, args action.Args) error {
			prj, err := s.Project()
			if err != nil {
				return err
			}
			return prj.Test(sess)
		})
}
