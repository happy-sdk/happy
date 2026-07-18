// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/happy-sdk/happy/addons/devel/pkg/gitutils"
	"github.com/happy-sdk/happy/addons/devel/pkg/gomodule"
	"github.com/happy-sdk/happy/addons/devel/pkg/views"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/session"
	tr "github.com/happy-sdk/lib/taskrunner"
)

func (prj *Project) Release(sess *session.Context, allowDirty, skipRemoteChecks bool) (err error) {

	if !prj.Config().Get("releaser.enabled").Value().Bool() {
		return errors.New("releasing is disabled")
	}
	releaser := tr.New("release")

	var previousTaskID tr.TaskID

	previousTaskID = prj.releaseAllowed(
		sess, releaser, allowDirty)

	// LINT
	lintTasks := prj.lintTasks(sess)
	linterEnabled := prj.Config().Get("linter.enabled").Value().Bool()

	for _, task := range lintTasks {
		if linterEnabled {
			task = task.DependsOn(previousTaskID)
			previousTaskID = task.ID()
		}
		releaser.AddTask(task)
	}

	// TEST
	releaser.Add("testing", func(ex *tr.Executor) (res tr.Result) {
		return tr.Success("running project tests...")
	})

	testTasks := prj.testTasks(sess)
	testsEnabled := prj.Config().Get("tests.enabled").Value().Bool()

	for _, task := range testTasks {
		if testsEnabled {
			task = task.DependsOn(previousTaskID)
			previousTaskID = task.ID()
		}
		releaser.AddTask(task)
	}

	releaser.AddD(previousTaskID, "commmit", func(ex *tr.Executor) (res tr.Result) {
		if gitutils.Dirty(sess, prj.Dir().Path, ".") {
			if err := gitutils.Commit(sess, prj.Dir().Path, []string{"-A"}, fmt.Sprintf("chore(%s): :label: prepare release", path.Base(prj.Dir().Path))); err != nil {
				return tr.Failure(err.Error())
			}
		}
		return tr.Skip("clean")
	})

	// GOMODULES
	previousTaskID, err = prj.releaseGomodules(sess, releaser, previousTaskID, skipRemoteChecks)
	if err != nil {
		return err
	}

	// CHANGELOG
	previousTaskID, err = prj.releaseChangelog(sess, releaser, previousTaskID)
	if err != nil {
		return err
	}

	// FINALIZE
	prj.releaseFinalize(sess, releaser, previousTaskID)
	return releaser.Run()
}

func (prj *Project) releaseAllowed(sess *session.Context, r *tr.Runner, allowDirty bool) tr.TaskID {
	t1 := r.Add("starting releaser", func(ex *tr.Executor) (res tr.Result) {
		gitDirty := gitutils.Dirty(sess, prj.Dir().Path, prj.Dir().Path)
		if gitDirty {
			msg := "project repository is dirty"
			if !allowDirty {
				return tr.Failure(msg)
			}
			return tr.Notice(msg)
		}
		return tr.Success("project repository clean")
	})

	t2 := r.AddD(t1, "checking git branch", func(ex *tr.Executor) (res tr.Result) {
		expectedBranch := prj.Config().Get("git.branch").String()
		currentBranch, err := gitutils.CurrentBranch(sess, prj.Dir().Path)
		if err != nil {
			return tr.Failure(err.Error())
		}
		if currentBranch != expectedBranch {
			return tr.Failure(fmt.Sprintf("expected branch %s, got %s", expectedBranch, currentBranch))
		}
		return tr.Success("ok")
	})

	t3 := r.AddD(t2, "checking git remote", func(ex *tr.Executor) (res tr.Result) {
		expectedRemoteURL := prj.Config().Get("git.remote.url").String()
		expectedRemoteName := prj.Config().Get("git.remote.name").String()
		currentRemoteName, currentRemoteURL, err := gitutils.CurrentRemote(sess, prj.Dir().Path)
		if err != nil {
			return tr.Failure(err.Error())
		}
		if currentRemoteURL != expectedRemoteURL {
			return tr.Failure(fmt.Sprintf("expected remote %s, got %s", expectedRemoteURL, currentRemoteURL))
		}
		if currentRemoteName != expectedRemoteName {
			return tr.Failure(fmt.Sprintf("expected remote name %s, got %s", expectedRemoteName, currentRemoteName))
		}
		return tr.Success("ok")
	})

	t4 := r.AddD(t3, "checking dist dir", func(ex *tr.Executor) (res tr.Result) {
		dist := prj.Dist()
		if dist == "" {
			return tr.Failure("dist dir not found")
		}
		if distStat, err := os.Stat(dist); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return tr.Failure(err.Error())
			}
			if err := os.Mkdir(dist, 0750); err != nil {
				return tr.Failure(err.Error())
			}
		} else if !distStat.IsDir() {
			return tr.Failure("dist is not a directory")
		}
		return tr.Success("ok").WithDesc(dist)
	})

	return t4
}

func (prj *Project) releaseFinalize(sess *session.Context, r *tr.Runner, prev tr.TaskID) {
	r.AddD(prev, "finalizing", func(ex *tr.Executor) (res tr.Result) {
		return tr.Success("release completed")
	})
}

func (prj *Project) releaseGomodules(sess *session.Context, r *tr.Runner, dep tr.TaskID, skipRemoteChecks bool) (tr.TaskID, error) {
	var (
		gomodules []*gomodule.Package
		err       error
		failed    bool
	)

	gomodules, err = prj.GoModules(sess)
	if err != nil {
		return dep, err
	}

	remoteName := prj.Config().Get("git.remote.name").String()

	var rootGoVersion string
	for _, pkg := range gomodules {
		if pkg.TagPrefix == "" && pkg.Modfile.Go != nil {
			rootGoVersion = pkg.Modfile.Go.Version
			break
		}
	}

	var goDepsLatest = make(map[string]version.Version)

	r.AddD(dep, "check dep updates",
		func(ex *tr.Executor) (res tr.Result) {
			goDepsToCheckLatest := prj.Config().Get("deps.go.latest").Value().Fields()

			if len(goDepsToCheckLatest) == 0 {
				return tr.Skip("no deps to check updates for")
			}

			for _, dep := range goDepsToCheckLatest {

				ver, err := gomodule.GetLatestVersion(dep, prj.Dir().Path)
				if err != nil {
					return tr.Failure(fmt.Sprintf("failed to get latest version for %s: %s", dep, err.Error()))
				}
				v, err := version.Parse(ver)
				if err != nil {
					return tr.Failure("failed to parse dep version").WithDesc(dep)
				}
				goDepsLatest[dep] = v
			}

			return tr.Success("deps checked")
		})

	for _, pkg := range gomodules {
		name := path.Base(pkg.Dir)
		r.AddD(dep, name,
			func(exs *tr.Executor) (res tr.Result) {
				if err := pkg.LoadReleaseInfo(
					sess,
					prj.Dir().Path,
					remoteName,
					!skipRemoteChecks); err != nil {
					failed = true
					return tr.Failure(fmt.Sprintf("failed to get release info: %s", err.Error()))
				}

				for goDepImport, goDepLatest := range goDepsLatest {
					for _, r := range pkg.Modfile.Require {
						if r.Mod.Path == goDepImport {
							if err := pkg.SetDep(goDepImport, goDepLatest); err != nil {
								return tr.Failure(fmt.Sprintf("failed to set dep %s: %s", goDepImport, err.Error()))
							}
							break
						}
					}
				}

				if pkg.TagPrefix != "" {
					if err := pkg.SyncGoVersion(rootGoVersion); err != nil {
						return tr.Failure(fmt.Sprintf("failed to sync go version: %s", err.Error()))
					}
				}

				if !pkg.NeedsRelease {
					return tr.Skip("no release needed").WithDesc(fmt.Sprintf("latest: %s", pkg.LastReleaseTag))
				}
				if pkg.PendingRelease {
					return tr.Skip(fmt.Sprintf("pending release %s -> %s", path.Base(pkg.LastReleaseTag), path.Base(pkg.NextReleaseTag))).WithDesc(pkg.Import)
				}
				return tr.Success(fmt.Sprintf("needs release %s", path.Base(pkg.NextReleaseTag))).WithDesc(pkg.Import)
			})
	}

	t1 := r.AddD(dep, "gomodules", func(exs *tr.Executor) (res tr.Result) {
		if failed {
			return tr.Failure("failed to load gomodules release info")
		}
		return tr.Success("gomodules release info loaded")
	})

	t2 := r.AddD(t1, "sort gomodules", func(exs *tr.Executor) (res tr.Result) {
		if _, err := gomodule.TopologicalReleaseQueue(gomodules); err != nil {
			return tr.Failure(fmt.Sprintf("failed to sort gomodules: %s", err.Error()))
		}
		return tr.Success("sorted releaseable gomodules")
	})

	var (
		commonDepsUpdated bool
		commonDeps        []gomodule.Dependency
	)
	t3 := r.AddD(t2, "check common go deps", func(ex *tr.Executor) (res tr.Result) {
		commonDeps, err = gomodule.GetCommonDeps(gomodules)
		if err != nil {
			return tr.Failure(fmt.Sprintf("failed to get common deps: %s", err.Error()))
		}

		return tr.Success(fmt.Sprintf("loaded common deps %d", len(commonDeps)))
	})

	t3_1 := r.AddD(t3, "update common go deps", func(ex *tr.Executor) (res tr.Result) {
		for _, dep := range commonDeps {
			ex.AddTick()
			if version.Compare(dep.MinVersion, dep.MaxVersion) != 0 {
				commonDepsUpdated = true
				for _, imprt := range dep.UsedBy {
					for _, pkg := range gomodules {
						if pkg.Import == imprt {
							name := path.Base(pkg.Dir)
							ex.Subtask(name, func(ex *tr.Executor) (res tr.Result) {
								if err := pkg.SetDep(dep.Import, dep.MaxVersion); err != nil {
									return tr.Failure(err.Error())
								}
								return tr.Success("updated").WithDesc(fmt.Sprintf("%s@%s", dep.Import, dep.MaxVersion))
							})
						}
					}
				}
			}
		}
		return tr.Success(fmt.Sprintf("loaded common deps %d", len(commonDeps)))
	})

	r.AddD(t3_1, "update go modules", func(ex *tr.Executor) (res tr.Result) {
		if !commonDepsUpdated {
			return tr.Skip("no deps updated")
		}
		return tr.Success("deps updated")
	})

	t4 := r.AddD(t3, "check modules to release", func(*tr.Executor) (res tr.Result) {
		count := 0
		for _, s := range gomodules {
			if s.NeedsRelease || s.PendingRelease {
				count++
			}
		}
		var msg string
		switch count {
		case 0:
			return tr.Success("no modules to release")
		case 1:
			msg = fmt.Sprintf("%d module", count)
		default:
			msg = fmt.Sprintf("%d modules", count)
		}
		return tr.Success(msg)
	})

	t5 := r.AddD(t4, "confirm releasable modules", func(ex *tr.Executor) (res tr.Result) {
		_ = ex.Program().ReleaseTerminal()
		defer func() {
			_ = ex.Program().RestoreTerminal()
		}()

		stdout := ex.Stdout()

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

	t6 := r.AddD(t5, "tag packages", func(ex *tr.Executor) (res tr.Result) {
		prevDep := t5

		for _, pkg := range gomodules {
			prevDep = pkg.ApplyTagTask(sess, ex, prevDep, prj.Dir().Path, gomodules)
		}
		return tr.Success("added package tag tasks")
	})

	return t6, nil
}

func (prj *Project) releaseChangelog(sess *session.Context, r *tr.Runner, dep tr.TaskID) (tr.TaskID, error) {
	t1 := r.AddD(dep, "changelog", func(exs *tr.Executor) (res tr.Result) {
		gomodules, err := prj.GoModules(sess)
		if err != nil {
			return tr.Failure(err.Error())
		}

		cl := &fullChangelog{}

		for _, pkg := range gomodules {
			if !pkg.NeedsRelease || (pkg.Changelog == nil) {
				continue
			}

			clp := &packageChangelog{pkg: pkg}

			for _, breaking := range pkg.Changelog.Breaking() {
				breaking := fmt.Sprintf("* %s %s", breaking.ShortHash, breaking.Subject)
				clp.Breaking = append(clp.Breaking, breaking)
			}

			for _, entry := range pkg.Changelog.Entries() {
				change := fmt.Sprintf("* %s %s", entry.ShortHash, entry.Subject)
				clp.Changes = append(clp.Changes, change)
			}

			if pkg.Dir == prj.Dir().Path {
				cl.Root = clp
			} else {
				cl.Subpkgs = append(cl.Subpkgs, clp)
			}
		}

		cldata := new(strings.Builder)
		cldata.WriteString("## Changelog\n")

		if cl.Root != nil {
			fmt.Fprintf(cldata, "`%s@%s`\n\n", cl.Root.pkg.Import, cl.Root.pkg.NextReleaseTag)
			var breakingsection string
			for _, breaking := range cl.Root.Breaking {
				for _, scl := range cl.Subpkgs {
					found := false
					for _, bcl := range scl.Breaking {
						if bcl == breaking {
							found = true
						}
					}
					if !found {
						breakingsection += breaking + "\n"
					}
				}
			}
			if len(breakingsection) > 0 {
				cldata.WriteString("### Breaking Changes\n")
				cldata.WriteString(breakingsection)
			}
			var changessection string
			for _, change := range cl.Root.Changes {
				found := false
				for _, scl := range cl.Subpkgs {
					found = slices.Contains(scl.Changes, change)
					if found {
						break
					}
				}
				if found {
					continue
				}
				changessection += change + "\n"
			}
			if len(changessection) > 0 {
				cldata.WriteString("### Changes\n")
				cldata.WriteString(changessection)
			}
			cldata.WriteString("\n")
		}

		for _, scl := range cl.Subpkgs {
			fmt.Fprintf(cldata, "\n### %s\n\n`%s@%s`\n", scl.pkg.NextReleaseTag, scl.pkg.Import, path.Base(scl.pkg.NextReleaseTag))

			for i, breaking := range scl.Breaking {
				if i == 0 {
					cldata.WriteString("**Breaking Changes**\n")
				}
				cldata.WriteString(breaking)
			}
			for i, change := range scl.Changes {
				if i == 0 {
					cldata.WriteString("**Changes**\n")
				}
				cldata.WriteString(change)
				cldata.WriteRune('\n')
			}
		}

		cldata.WriteString("\n")
		clFilePath := filepath.Join(prj.Dist(), "CHANGELOG.md")
		if err := os.WriteFile(clFilePath, []byte(cldata.String()), 0644); err != nil {
			return tr.Failure(err.Error())
		}

		return tr.Success("changelog saved")
	})
	return t1, nil
}

type fullChangelog struct {
	Root    *packageChangelog
	Subpkgs []*packageChangelog
}

type packageChangelog struct {
	pkg      *gomodule.Package
	Breaking []string
	Changes  []string
}
