// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package tasks

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects/project"
	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects/views"
	"github.com/happy-sdk/happy/cmd/gohappy/pkg/git"
	"github.com/happy-sdk/happy/cmd/gohappy/pkg/gomodule"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/session"
	"github.com/happy-sdk/taskrunner"
	tr "github.com/happy-sdk/taskrunner"
)

// Check Releaseables
func GoModulesCheck(
	r *tr.Runner,
	prj *project.Project,
	sess *session.Context,
	dep tr.TaskID,
) tr.TaskID {
	var gomodules []*gomodule.Package

	// t1
	t1 := r.AddD(dep, "get go modules", func(ex *tr.Executor) (res tr.Result) {
		var err error
		gomodules, err = prj.GoModules(sess, false, false)
		if err != nil {
			return taskrunner.Failure(fmt.Sprintf("failed to get modules: %s", err.Error()))
		}
		for _, pkg := range gomodules {
			name := path.Base(pkg.Dir)
			wd := prj.Config().Get("git.repo.root").String()
			origin := prj.Config().Get("git.repo.remote.name").String()

			ex.Subtask(fmt.Sprintf("%s: get release info", name),
				func(exs *tr.Executor) (res tr.Result) {
					if err := pkg.LoadReleaseInfo(sess, wd, origin, true); err != nil {
						return taskrunner.Failure(fmt.Sprintf("failed to get release info: %s", err.Error()))
					}
					return taskrunner.Success("got release info")
				})
		}

		ex.Subtask("sort releases", func(*tr.Executor) (res tr.Result) {
			if err := prj.UpdateTopologicalReleaseQueue(); err != nil {
				return tr.Failure(err.Error())
			}
			return tr.Success("sorted")
		})

		return taskrunner.Success(fmt.Sprintf("loaded modules %d", len(gomodules)))
	})

	// t2
	var commonDepsUpdated bool
	t2 := r.AddD(t1, "check common go deps", func(ex *tr.Executor) (res tr.Result) {
		commonDeps, err := gomodule.GetCommonDeps(gomodules)
		if err != nil {
			return taskrunner.Failure(fmt.Sprintf("failed to get common deps: %s", err.Error()))
		}
		for _, dep := range commonDeps {
			ex.AddTick()
			if version.Compare(dep.MinVersion, dep.MaxVersion) != 0 {
				commonDepsUpdated = true
				for _, imprt := range dep.UsedBy {
					ex.AddTick()
					for _, pkg := range gomodules {
						ex.AddTick()
						if pkg.Import == imprt {
							if err := pkg.SetDep(dep.Import, dep.MaxVersion); err != nil {
								return taskrunner.Failure(err.Error())
							}
						}
					}
				}
			}
		}
		return taskrunner.Success(fmt.Sprintf("loaded common deps %d", len(commonDeps)))
	})

	// t3
	t3 := r.AddD(t2, "update releaseable go modules", func(ex *tr.Executor) (res tr.Result) {
		if !commonDepsUpdated {
			return tr.Success("no deps updated")
		}
		var err error
		gomodules, err = prj.GoModules(sess, false, false)
		if err != nil {
			return taskrunner.Failure(err.Error())
		}
		return tr.Success("go modules reloaded")
	})

	// t4
	t4 := r.AddD(t3, "check modules to release", func(*tr.Executor) (res tr.Result) {
		return taskCheckModulesToRelease(gomodules)
	})

	return t4
}

func taskCheckModulesToRelease(gomodules []*gomodule.Package) taskrunner.Result {
	count := 0
	for _, s := range gomodules {
		if s.NeedsRelease {
			count++
		}
	}
	var msg string
	if count == 0 {
		return taskrunner.Success("no modules to release")
	} else if count == 1 {
		msg = fmt.Sprintf("%d module", count)
	} else {
		msg = fmt.Sprintf("%d modules", count)
	}
	return taskrunner.Success(msg)
}

// Confirm Releaseables
func GoModulesConfirm(
	r *tr.Runner,
	prj *project.Project,
	sess *session.Context,
	dep tr.TaskID,
) tr.TaskID {
	return r.AddD(dep, "confirm releasable modules", func(ex *tr.Executor) (res tr.Result) {
		ex.Program().ReleaseTerminal()
		defer ex.Program().RestoreTerminal()

		stdout := ex.Stdout()
		gomodules, err := prj.GoModules(sess, false, false)
		if err != nil {
			return tr.Failure(err.Error())
		}

		view, err := views.GetConfirmReleasablesView(sess, gomodules)
		if err != nil {
			return tr.Failure(err.Error())
		}

		m, err := tea.NewProgram(
			view,
			tea.WithOutput(stdout),
			tea.WithAltScreen(),
		).Run()
		if err != nil {
			fmt.Println("Error running program:", err)
		}

		model, ok := m.(views.ConfirmReleasablesView)
		if !ok {
			return tr.Failure("could not assert model type")
		}
		if !model.Yes {
			return tr.Failure("user did not confirm release")
		}
		return tr.Success("continue with release")
	})
}

// Prepare Releaseables
func GoModulesPrepare(
	r *tr.Runner,
	prj *project.Project,
	sess *session.Context,
	dep tr.TaskID,
) tr.TaskID {
	return r.AddD(dep, "prepare go mod releases", func(ex *tr.Executor) (res tr.Result) {
		gomodules, err := prj.GoModules(sess, false, false)
		if err != nil {
			return tr.Failure(err.Error())
		}
		for _, gomod := range gomodules {
			goModulesPrepareReleaseable(ex, prj, sess, gomodules, gomod)
		}

		return tr.Success("go modules prepared")
	})
}

func goModulesPrepareReleaseable(
	ex *tr.Executor,
	prj *project.Project,
	sess *session.Context,
	gomodules []*gomodule.Package,
	gomod *gomodule.Package,
) tr.TaskID {

	internalDeps := make(map[string]*gomodule.Package)
	for _, pkg := range gomodules {
		internalDeps[pkg.Import] = pkg
	}

	name := path.Base(gomod.Dir)

	// Check does module need release
	t1 := ex.Subtask(fmt.Sprintf("%s: need release", name),
		func(exs *tr.Executor) tr.Result {
			if !gomod.NeedsRelease {
				return tr.Skip(gomod.LastReleaseTag).WithDesc(gomod.Import)
			} else if gomod.PendingRelease {
				return tr.Success(fmt.Sprintf("pending release %s", gomod.NextReleaseTag)).WithDesc(gomod.Import)
			}
			msg := fmt.Sprintf("%s%s -> %s",
				gomod.TagPrefix,
				path.Base(gomod.LastReleaseTag),
				path.Base(gomod.NextReleaseTag),
			)
			if gomod.FirstRelease {
				msg = fmt.Sprintf("%s%s",
					gomod.TagPrefix,
					path.Base(gomod.NextReleaseTag),
				)
			}
			return tr.Success(msg).WithDesc(gomod.Import)
		},
	)

	var monorepoDeps []string
	t2 := ex.SubtaskD(t1, fmt.Sprintf("%s: verify dependencies", name),
		func(exs *tr.Executor) tr.Result {
			if gomod.PendingRelease {
				return tr.Success(fmt.Sprintf("pending release %s", gomod.NextReleaseTag)).WithDesc(gomod.Import)
			}
			for _, require := range gomod.Modfile.Require {
				if dep, ok := internalDeps[require.Mod.Path]; ok {
					if dep.NeedsRelease {
						if !git.TagExists(sess, prj.WD(), dep.NextReleaseTag) {
							return tr.Failure(fmt.Sprintf("tag %s does not exist", dep.NextReleaseTag))
						}
						monorepoDeps = append(monorepoDeps, dep.Import)
						relPath, err := filepath.Rel(prj.WD(), dep.Dir)
						if err != nil {
							return tr.Failure("calculate relative path").WithDesc(err.Error())
						}
						if err := gomod.Modfile.AddReplace(dep.Import, "", relPath, ""); err != nil {
							return tr.Failure("add tmp replace").WithDesc(err.Error())
						}
					}
				}
			}
			return tr.Success("dependencies verified")
		},
	)

	t3 := ex.SubtaskD(t2, fmt.Sprintf("%s: update go.mod", name),
		func(exs *tr.Executor) tr.Result {
			if gomod.PendingRelease {
				return tr.Success(fmt.Sprintf("pending release %s", gomod.NextReleaseTag)).WithDesc(gomod.Import)
			}
			gomod.Modfile.Cleanup()
			updatedModFile, err := gomod.Modfile.Format()
			if err != nil {
				return tr.Failure("format go.mod").WithDesc(err.Error())
			}

			if err := os.WriteFile(gomod.ModFilePath, updatedModFile, 0644); err != nil {
				return tr.Failure("write go.mod").WithDesc(err.Error())
			}

			if err := gomod.Tidy(sess); err != nil {
				return tr.Failure("tidy go.mod").WithDesc(err.Error())
			}

			if len(monorepoDeps) > 0 {
				for _, dep := range monorepoDeps {
					if err := gomod.Modfile.DropReplace(dep, ""); err != nil {
						return tr.Failure("drop replace").WithDesc(err.Error())
					}
				}

				gomod.Modfile.Cleanup()
				updatedModFile, err := gomod.Modfile.Format()
				if err != nil {
					return tr.Failure("format go.mod after drop replace").WithDesc(err.Error())
				}
				if err := os.WriteFile(gomod.ModFilePath, updatedModFile, 0644); err != nil {
					return tr.Failure("write go.mod after drop replace").WithDesc(err.Error())
				}
			}
			return tr.Success("go.mod updated")
		},
	)

	t4 := ex.SubtaskD(t3, fmt.Sprintf("%s: commit", name),
		func(exs *tr.Executor) tr.Result {
			if gomod.PendingRelease {
				return tr.Success(fmt.Sprintf("pending release %s", gomod.NextReleaseTag)).WithDesc(gomod.Import)
			}
			wd := prj.WD()
			p, err := filepath.Rel(wd, gomod.Dir)
			if err != nil {
				return tr.Failure("relative path").WithDesc(err.Error())
			}

			if !git.Dirty(sess, wd, p) {
				return tr.Success("git path clean").WithDesc(p)
			}

			msg := fmt.Sprintf("deps(%s): :label: prepare release %s", name, path.Base(gomod.NextReleaseTag))
			if err := git.Commit(sess, wd, p, msg); err != nil {
				return tr.Failure("commit").WithDesc(err.Error())
			}

			return tr.Success(p)
		})

	t5 := ex.SubtaskD(t4, fmt.Sprintf("%s: tag", name),
		func(exs *tr.Executor) tr.Result {
			if gomod.PendingRelease {
				return tr.Success(fmt.Sprintf("pending release %s", gomod.NextReleaseTag)).WithDesc(gomod.Import)
			}
			return tr.Success(gomod.NextReleaseTag)
		})
	return t5
}
