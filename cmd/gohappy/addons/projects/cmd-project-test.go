// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"github.com/happy-sdk/taskrunner"
	tr "github.com/happy-sdk/taskrunner"
)

func cmdProjectTest() *command.Command {
	return command.New("test",
		command.Config{
			Description: "Run project tests",
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
			if err := prj.Load(sess); err != nil {
				return err
			}

			gomodules, err := prj.GoModules(sess, false, false)
			if err != nil {
				return err
			}

			runner := tr.New()

			output := make(map[string]string)

			for _, gomodule := range gomodules {

				runner.Add(filepath.Base(gomodule.Dir), func(*tr.Executor) (res tr.Result) {
					// Get packages belonging to module
					localPkgsCmd := exec.Command("go", "list", "./...")
					localPkgsCmd.Dir = gomodule.Dir
					localPkgsOut, err := cli.ExecRaw(sess, localPkgsCmd)
					if err != nil {
						return taskrunner.Failure(err.Error()).WithDesc(gomodule.Import)
					}

					localPkgs := strings.Join(strings.Fields(string(localPkgsOut)), ",")

					testCmd := exec.Command("go", "test", "-coverpkg", localPkgs, "-coverprofile", "coverage.out", "-timeout", "1m", "./...")
					testCmd.Dir = gomodule.Dir

					out, err := cli.Exec(sess, testCmd)
					if err != nil {
						fmt.Println(out)
						return taskrunner.Failure(err.Error()).WithDesc(gomodule.Import)
					}

					coverageSumCmd := exec.Command("go", "tool", "cover", "-func", "coverage.out")
					coverageSumCmd.Dir = gomodule.Dir

					coverageSumOut, err := cli.Exec(sess, coverageSumCmd)
					if err != nil {
						output[gomodule.Import] = coverageSumOut
						return taskrunner.Failure(err.Error()).WithDesc(gomodule.Import)
					}

					lines := strings.Split(strings.TrimSpace(string(coverageSumOut)), "\n")
					var coverage vars.Value
					if len(lines) > 0 {
						lastLine := lines[len(lines)-1]

						cov, err := testutils.ExtractCoverage(lastLine)
						if err != nil {
							return taskrunner.Failure(err.Error()).WithDesc(gomodule.Import)
						}
						coverage, _ = vars.NewValue(strings.TrimSuffix(cov, "%"))
					}
					c, _ := coverage.Float64()
					if c == 100.0 {
						return taskrunner.Success(fmt.Sprintf("coverage[ %-8s]: full", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
					} else if c >= 90.0 {
						return taskrunner.Success(fmt.Sprintf("coverage[ %-8s]: high", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
					} else if c >= 75.0 {
						return taskrunner.Info(fmt.Sprintf("coverage[ %-8s]: moderate", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
					} else if c >= 50.0 {
						return taskrunner.Notice(fmt.Sprintf("coverage[ %-8s]: low", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
					} else if c > 0.0 {
						return taskrunner.Warn(fmt.Sprintf("coverage[ %-8s]: very-low", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
					} else {
						return taskrunner.Warn("coverage[ 0%      ]: no coverage").WithDesc(gomodule.Import)
					}
				})

			}

			if err := runner.Run(); err != nil {
				return err
			}

			for gomodule, out := range output {
				fmt.Println(gomodule)
				fmt.Println("-------------------------------------------")
				fmt.Println(out)
				fmt.Println("-------------------------------------------")
			}

			if len(output) > 0 {
				return errors.New("test errors")
			}

			return nil
		})
}
