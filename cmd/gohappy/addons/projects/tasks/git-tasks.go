// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package tasks

import (
	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects/project"
	tr "github.com/happy-sdk/taskrunner"
)

func GitCheckDirty(r *tr.Runner, prj *project.Project, allowDirty bool) tr.TaskID {
	return r.Add("git dirty", func(ex *tr.Executor) (res tr.Result) {
		if prj.Dirty() {
			if allowDirty {
				return tr.Notice("releasing dirty git repo")
			}
			return tr.Failure("project git repository is dirty")
		}
		return tr.Success("ok")
	})
}
