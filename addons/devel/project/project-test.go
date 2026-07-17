// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/pkg/devel/testutils"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
	tr "github.com/happy-sdk/lib/taskrunner"
)

func (prj *Project) Test(sess *session.Context) error {

	testsuite := tr.New("test")

	tasks := prj.testTasks(sess)

	for _, t := range tasks {
		testsuite.AddTask(t)
	}

	if err := testsuite.Run(); err != nil {
		return err
	}

	return nil
}

func (prj *Project) testTasks(sess *session.Context) []tr.Task {
	var tasks []tr.Task

	if !prj.Config().Get("tests.enabled").Value().Bool() {
		tasks = append(tasks, tr.NewTask("tests", func(ex *tr.Executor) (res tr.Result) {
			return tr.Skip("tests disabled")
		}))
		return tasks
	}

	tasks = append(tasks, tr.NewTask("tests", func(ex *tr.Executor) (res tr.Result) {
		return tr.Success("tests enabled")
	}))

	gomodules, err := prj.GoModules(sess)
	if err != nil {
		tasks = append(tasks, tr.NewTask("listing go modules", func(ex *tr.Executor) (res tr.Result) {
			return tr.Failure("failed to list go modules").
				WithDesc(err.Error())
		}))
		return tasks
	}

	for _, gomodule := range gomodules {
		name := gomodule.TagPrefix
		if gomodule.TagPrefix == "" {
			name = filepath.Base(gomodule.Dir)
		}

		// Respect config ignore list (from .happy.yaml: ignore: [])
		if prj.isIgnoredModule(gomodule.Dir) {
			continue
		}

		t := tr.NewTask(name, func(ex *tr.Executor) (res tr.Result) {
			// Get packages belonging to module
			localPkgsCmd := exec.Command("go", "list", "./...")
			localPkgsCmd.Dir = gomodule.Dir
			localPkgsOut, err := cli.ExecRaw(sess, localPkgsCmd)
			if err != nil {
				return tr.Failure(err.Error()).WithDesc(gomodule.Import)
			}

			localPkgs := strings.Join(strings.Fields(string(localPkgsOut)), ",")

			testCmd := exec.Command("go", "test", "-race", "-coverpkg", localPkgs, "-coverprofile", "coverage.out", "-timeout", "1m", "./...")
			testCmd.Dir = gomodule.Dir

			out, err := cli.Exec(sess, testCmd)
			if err != nil {
				ex.Println(out)
				return tr.Failure(err.Error()).WithDesc(gomodule.Import)
			}

			coverageSumCmd := exec.Command("go", "tool", "cover", "-func", "coverage.out")
			coverageSumCmd.Dir = gomodule.Dir

			coverageSumOut, err := cli.Exec(sess, coverageSumCmd)
			if err != nil {
				ex.Println(coverageSumOut)
				return tr.Failure(err.Error()).WithDesc(gomodule.Import)
			}

			lines := strings.Split(strings.TrimSpace(string(coverageSumOut)), "\n")
			var coverage vars.Value
			if len(lines) > 0 {
				lastLine := lines[len(lines)-1]

				cov, err := testutils.ExtractCoverage(lastLine)
				if err != nil {
					return tr.Failure(err.Error()).WithDesc(gomodule.Import)
				}
				coverage, _ = vars.NewValue(strings.TrimSuffix(cov, "%"))
			}
			c, _ := coverage.Float64()
			if c == 100.0 {
				return tr.Success(fmt.Sprintf("coverage[ %-8s]: full", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
			} else if c >= 90.0 {
				return tr.Success(fmt.Sprintf("coverage[ %-8s]: high", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
			} else if c >= 75.0 {
				return tr.Info(fmt.Sprintf("coverage[ %-8s]: moderate", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
			} else if c >= 50.0 {
				return tr.Notice(fmt.Sprintf("coverage[ %-8s]: low", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
			} else if c > 0.0 {
				return tr.Warn(fmt.Sprintf("coverage[ %-8s]: very-low", coverage.FormatFloat('f', 2, 64)+"%")).WithDesc(gomodule.Import)
			} else {
				return tr.Warn("coverage[ 0%      ]: no coverage").WithDesc(gomodule.Import)
			}
		})

		tasks = append(tasks, t)
	}

	return tasks
}
