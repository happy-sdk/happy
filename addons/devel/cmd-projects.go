// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

import (
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func cmdProjects() *command.Command {
	return command.New("projects",
		command.Config{
			Description: "Manage local projects known by Happy SDK",
		}).
		Do(func(sess *session.Context, args action.Args) error {
			sess.Log().NotImplemented("projects command not implemented")
			return nil
		})
}
