// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
	"github.com/happy-sdk/taskrunner"
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

			repoRoot := prj.Config().Get("git.repo.root").String()
			runner := taskrunner.New("project tests")

			output := make(map[string]string)

			for _, gomodule := range gomodules {

				testGroup := taskrunner.NewGroup(gomodule.Import)

				// Get packages belonging to module
				localPkgsCmd := exec.Command("go", "list", "./...")
				localPkgsCmd.Dir = gomodule.Dir
				localPkgsOut, err := cli.ExecRaw(sess, localPkgsCmd)
				if err != nil {
					return err
				}
				for localPkg := range strings.SplitSeq(string(localPkgsOut), "\n") {
					if localPkg == "" {
						continue
					}
					name := strings.TrimPrefix(localPkg, gomodule.Import)
					if name == "" {
						name = gomodule.TagPrefix
						if gomodule.TagPrefix == "" {
							name = "."
						}
					}
					if strings.HasPrefix(name, "/") {
						name = filepath.Join(gomodule.TagPrefix, name)
					}

					testGroup.Task(name, func() (res taskrunner.Result) {

						wd := filepath.Join(repoRoot, name)
						testCmd := exec.Command("go", "test", "-coverprofile", "coverage.out", "-timeout", "1m", ".")
						testCmd.Dir = wd
						out, err := cli.Exec(sess, testCmd)
						if err != nil {
							output[gomodule.Import] = out
							return taskrunner.Failure(err.Error(), "test errors")
						}
						coverage, err := extractCoverage(out)
						if err != nil {
							return taskrunner.Failure(err.Error(), "coverage extraction failed")
						}
						if coverage == "no test files" || coverage == "0.0%" {
							return taskrunner.Warn(coverage, "no coverage")
						}
						return taskrunner.Success(coverage, "")
					})

				}

				if err := runner.Add(testGroup); err != nil {
					return err
				}

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
				return errors.New("test errors")
			}

			return nil
		})
}

func extractCoverage(s string) (string, error) {
	// Match coverage percentage
	covRe := regexp.MustCompile(`coverage: (\d+\.\d+%)`)
	if match := covRe.FindStringSubmatch(s); len(match) > 1 {
		return match[1], nil
	}

	// Match "no test files"
	noTestRe := regexp.MustCompile(`\[no test files\]`)
	if noTestRe.MatchString(s) {
		return "no test files", nil
	}

	return "", fmt.Errorf("no coverage or test files info found")
}
