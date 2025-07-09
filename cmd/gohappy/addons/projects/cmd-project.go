// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func cmdProject() *command.Command {
	return command.New("project",
		command.Config{
			Description:        "Manage current project",
			SharedBeforeAction: true,
			FailDisabled:       true,
		}).
		Disable(func(sess *session.Context) error {
			api, err := happy.API[*API](sess)
			if err != nil {
				return err
			}

			_, err = api.Project(sess, false)
			return err
		}).
		WithSubCommands(
			cmdProjectInfo(),
			cmdProjectLint(),
			cmdProjectRelease(),
			cmdProjectTest(),
		)
}
