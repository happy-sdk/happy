// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func cmdProjectRelease() *command.Command {
	return command.New("release",
		command.Config{
			Description: "Release current project",
		}).
		Do(func(sess *session.Context, args action.Args) error {
			api, err := happy.API[*API](sess)
			if err != nil {
				return err
			}
			prj, err := api.Project()
			if err != nil {
				return err
			}
			if err := prj.Load(sess); err != nil {
				return err
			}
			return api.ProjectInfoPrint(sess)
		})
}
