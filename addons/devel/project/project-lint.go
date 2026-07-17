// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
	tr "github.com/happy-sdk/lib/taskrunner"
)

func (prj *Project) Lint(sess *session.Context) error {

	if !prj.Config().Get("linter.enabled").Value().Bool() {
		return fmt.Errorf("%w: linting disabled", Error)
	}

	linter := tr.New("lint")
	tasks := prj.lintTasks(sess)
	for _, t := range tasks {
		linter.AddTask(t)
	}

	return linter.Run()
}

func (prj *Project) lintTasks(sess *session.Context) []tr.Task {
	var tasks []tr.Task

	if !prj.Config().Get("linter.enabled").Value().Bool() {
		tasks = append(tasks, tr.NewTask("linting", func(ex *tr.Executor) (res tr.Result) {
			return tr.Skip("linting disabled")
		}))
		return tasks
	}

	tasks = append(tasks, tr.NewTask("linting", func(ex *tr.Executor) (res tr.Result) {
		return tr.Success("linting enabled")
	}))

	gomodules, err := prj.GoModules(sess)
	if err != nil {
		tasks = append(tasks, tr.NewTask("listing go modules", func(ex *tr.Executor) (res tr.Result) {
			return tr.Failure("failed to list go modules").
				WithDesc(err.Error())
		}))
		return tasks
	}

	if prj.Config().Get("linter.golangci-lint.enabled").Value().Bool() {
		gloangciLintBin := prj.Config().Get("linter.golangci-lint.path").String()
		for _, gomodule := range gomodules {
			// Respect config ignore list (from .happy.yaml: ignore: [])
			if prj.isIgnoredModule(gomodule.Dir) {
				continue
			}
			name := gomodule.TagPrefix
			if gomodule.TagPrefix == "" {
				name = filepath.Base(gomodule.Dir)
			}
			t := tr.NewTask(name, func(ex *tr.Executor) (res tr.Result) {
				cmd := exec.Command(gloangciLintBin, "run", "./...")
				cmd.Dir = gomodule.Dir
				out, err := cli.Exec(sess, cmd)
				if err != nil {
					ex.Println(out)
					return tr.Failure(err.Error()).WithDesc(gomodule.Import)
				}
				return tr.Success("ok").WithDesc(gomodule.Import)
			})
			tasks = append(tasks, t)
		}
	}

	return tasks
}

// isIgnoredModule reports whether the given module directory should be
// skipped based on the project's "ignore" list from .happy.yaml.
func (prj *Project) isIgnoredModule(modDir string) bool {
	ignores := prj.Config().Get("ignore").Value().Fields()
	if len(ignores) == 0 {
		return false
	}

	// Compute module path relative to project root for matching.
	rel, err := filepath.Rel(prj.dir.Path, modDir)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)

	for _, pat := range ignores {
		if pat == "" {
			continue
		}
		pat = filepath.ToSlash(pat)
		if rel == pat || strings.HasPrefix(rel, pat+"/") {
			return true
		}
	}
	return false
}
