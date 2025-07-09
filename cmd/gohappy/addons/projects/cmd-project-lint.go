// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"github.com/happy-sdk/taskrunner"
	tr "github.com/happy-sdk/taskrunner"
)

func cmdProjectLint() *command.Command {
	return command.New("lint",
		command.Config{
			Description: "Lint current project",
		}).
		Do(func(sess *session.Context, args action.Args) error {
			api, err := happy.API[*API](sess)
			if err != nil {
				return err
			}
			prj, err := api.Project(sess, true)
			if err != nil {
				return err
			}
			gomodules, err := prj.GoModules(sess, false, false)
			if err != nil {
				return err
			}

			gloangciLintBin := "golangci-lint"
			gloangciLintBinGh, isSet := os.LookupEnv("GITHUB_WORKSPACE")
			if isSet {
				gloangciLintBin = filepath.Join(gloangciLintBinGh, "bin", gloangciLintBin)
			}

			runner := tr.New()

			for _, gomodule := range gomodules {
				runner.Add(filepath.Base(gomodule.Dir), func(*tr.Executor) (res tr.Result) {
					cmd := exec.Command(gloangciLintBin, "run", "./...")
					cmd.Dir = gomodule.Dir
					out, err := cli.Exec(sess, cmd)
					if err != nil {
						fmt.Println(out)
						return taskrunner.Failure(err.Error()).WithDesc(gomodule.Import)
					}
					return taskrunner.Success("ok").WithDesc(gomodule.Import)
				})
			}

			if err := runner.Run(); err != nil {
				return err
			}

			return nil
		})
}
