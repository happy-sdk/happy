// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/happy-sdk/happy/addons/devel/pkg/gomodule"
	tr "github.com/happy-sdk/happy/lib/taskrunner"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/session"
)

// verifyEnv is the environment a verify task runs in: go.work disabled, and
// the module proxy/checksum database bypassed in favor of fetching straight
// from each module's VCS remote. GOWORK=off is the whole point - every
// earlier step in Release (lint, test, go mod tidy) runs inside the
// workspace, so it resolves sibling imports to on-disk source regardless of
// the version pinned in a module's own go.mod. GOPROXY=direct and
// GOSUMDB=off skip proxy.golang.org and sum.golang.org, which can lag a tag
// pushed moments earlier; this check is meant to give an immediate answer
// using the tag that was just pushed to the configured remote.
func verifyEnv() []string {
	return append(os.Environ(), "GOWORK=off", "GOPROXY=direct", "GOSUMDB=off")
}

// verifySkip reports whether pkg should be skipped by the verify step, and
// why. Only modules actually released this run need verifying - unreleased
// modules and internal-only packages have nothing new to prove.
func verifySkip(pkg *gomodule.Package) (bool, string) {
	if pkg.IsInternal {
		return true, "internal package"
	}
	if !pkg.NeedsRelease {
		return true, "not released this run"
	}
	return false, ""
}

func (prj *Project) verifyTasks(sess *session.Context) []tr.Task {
	var tasks []tr.Task

	if !prj.Config().Get("releaser.verify.enabled").Value().Bool() {
		tasks = append(tasks, tr.NewTask("verify", func(ex *tr.Executor) (res tr.Result) {
			return tr.Skip("verify disabled")
		}))
		return tasks
	}

	tasks = append(tasks, tr.NewTask("verify", func(ex *tr.Executor) (res tr.Result) {
		return tr.Success("verifying released modules build against pushed tags")
	}))

	gomodules, err := prj.GoModules(sess)
	if err != nil {
		tasks = append(tasks, tr.NewTask("listing go modules", func(ex *tr.Executor) (res tr.Result) {
			return tr.Failure("failed to list go modules").
				WithDesc(err.Error())
		}))
		return tasks
	}

	for _, pkg := range gomodules {
		// Respect config ignore list (from .happy.yaml: ignore: [])
		if prj.isIgnoredModule(pkg.Dir) {
			continue
		}

		name := pkg.TagPrefix
		if pkg.TagPrefix == "" {
			name = filepath.Base(pkg.Dir)
		}

		t := tr.NewTask(name, func(ex *tr.Executor) (res tr.Result) {
			if skip, reason := verifySkip(pkg); skip {
				return tr.Skip(reason).WithDesc(pkg.Import)
			}

			env := verifyEnv()

			buildCmd := exec.Command("go", "build", "./...")
			buildCmd.Dir = pkg.Dir
			buildCmd.Env = env
			if out, err := cli.Exec(sess, buildCmd); err != nil {
				ex.Println(out)
				return tr.Failure(err.Error()).WithDesc(pkg.Import)
			}

			testCmd := exec.Command("go", "test", "-count=1", "./...")
			testCmd.Dir = pkg.Dir
			testCmd.Env = env
			out, err := cli.Exec(sess, testCmd)
			if err != nil {
				ex.Println(out)
				return tr.Failure(err.Error()).WithDesc(pkg.Import)
			}

			return tr.Success(fmt.Sprintf("built and tested against %s", path.Base(pkg.NextReleaseTag))).WithDesc(pkg.Import)
		})
		tasks = append(tasks, t)
	}

	return tasks
}
