// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects/tasks"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	tr "github.com/happy-sdk/taskrunner"
)

func cmdProjectRelease() *command.Command {
	return command.New("release",
		command.Config{
			Description: "Release current project",
		}).
		WithFlags(
			cli.NewBoolFlag("dirty", false, "allow release from dirty git repository"),
		).
		Do(func(sess *session.Context, args action.Args) error {
			api, err := happy.API[*API](sess)
			if err != nil {
				return err
			}
			prj, err := api.Project(sess, true)
			if err != nil {
				return err
			}

			runner := tr.New()

			dep1 := tasks.GitCheckDirty(runner, prj, args.Flag("dirty").Var().Bool())
			dep2 := tasks.GoModulesCheck(runner, prj, sess, dep1)
			dep3 := tasks.GoModulesConfirm(runner, prj, sess, dep2)
			dep4 := tasks.GoModulesPrepare(runner, prj, sess, dep3)

			runner.AddD(dep4, "finalize", func(ex *tr.Executor) (res tr.Result) {
				return tr.Success("relaesed")
			})
			return runner.Run()
		})
}
