// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"errors"
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
			prj, err := api.Project()
			if err != nil {
				return err
			}
			if err := prj.Load(sess); err != nil {
				return err
			}
			gomodules, err := prj.GoModules(sess, false)
			if err != nil {
				return err
			}

			gloangciLintBin := "golangci-lint"
			gloangciLintBinGh, isSet := os.LookupEnv("GITHUB_WORKSPACE")
			if isSet {
				gloangciLintBin = filepath.Join(gloangciLintBinGh, "bin", gloangciLintBin)
			}

			runner := taskrunner.New("project linter")
			lintGoModules := taskrunner.NewGroup("lint go modules")

			output := make(map[string]string)

			for _, gomodule := range gomodules {
				lintGoModules.Task(gomodule.Import, func() (res taskrunner.Result) {
					cmd := exec.Command(gloangciLintBin, "run", "./...")
					cmd.Dir = gomodule.Dir
					out, err := cli.Exec(sess, cmd)
					if err != nil {
						output[gomodule.Import] = out
						return taskrunner.Failure(err.Error(), "linting errors")
					}
					return taskrunner.Success("ok", "")
				})
			}

			if err := runner.Add(lintGoModules); err != nil {
				return err
			}

			if err := runner.Run("*"); err != nil {
				return err
			}

			for gomodule, out := range output {
				fmt.Println(gomodule)
				fmt.Println("-------------------------------------------")
				fmt.Println(out)
				fmt.Println("-------------------------------------------")
			}

			if len(output) > 0 {
				return errors.New("linting errors")
			}

			return nil
		})
}
