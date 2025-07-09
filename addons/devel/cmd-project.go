// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
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

			_, err = api.Open(sess, sess.Get("app.fs.path.wd").String())
			return err
		}).
		WithSubCommands(
			cmdProjectInfo(),
			cmdProjectLint(),
			cmdProjectRelease(),
			cmdProjectTest(),
		)
}

func cmdProjectInfo() *command.Command {
	return command.New("info",
		command.Config{
			Description: "Print info about current project",
		}).
		Do(func(sess *session.Context, args action.Args) error {
			sess.Log().NotImplemented("info command not implemented")
			return nil
		})
}

func cmdProjectLint() *command.Command {
	return command.New("lint",
		command.Config{
			Description: "Lint current project",
		}).
		Do(func(sess *session.Context, args action.Args) error {
			sess.Log().NotImplemented("lint command not implemented")
			return nil
		})
}

func cmdProjectRelease() *command.Command {
	return command.New("release",
		command.Config{
			Description: "Release current project",
		}).
		WithFlags(
			cli.NewBoolFlag("dirty", false, "allow release from dirty git repository"),
		).
		Do(func(sess *session.Context, args action.Args) error {
			sess.Log().NotImplemented("test command not implemented")
			return nil
		})
}

func cmdProjectTest() *command.Command {
	return command.New("test",
		command.Config{
			Description: "Run project tests",
		}).
		Do(func(sess *session.Context, args action.Args) error {
			sess.Log().NotImplemented("test command not implemented")
			return nil
		})
}
