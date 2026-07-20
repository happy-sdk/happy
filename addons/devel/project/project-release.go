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
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/happy-sdk/happy/addons/devel/pkg/gitutils"
	"github.com/happy-sdk/happy/addons/devel/pkg/gomodule"
	"github.com/happy-sdk/happy/addons/devel/pkg/views"
	"github.com/happy-sdk/happy/lib/changelog"
	tr "github.com/happy-sdk/happy/lib/taskrunner"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/session"
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

	t4 := r.AddD(t3, "checking build dir", func(ex *tr.Executor) (res tr.Result) {
		dist := prj.Dist()
		if dist == "" {
			return tr.Failure("build dir not found")
		}
		if distStat, err := os.Stat(dist); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return tr.Failure(err.Error())
			}
			if err := os.MkdirAll(dist, 0750); err != nil {
				return tr.Failure(err.Error())
			}
		} else if !distStat.IsDir() {
			return tr.Failure("build dir is not a directory")
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
		return prj.writeChangelogs(gomodules)
	})
	return t1, nil
}

// writeChangelogs decides, based on whether this repo has a root Go module,
// whether to write one unified changelog (writeRootChangelog) or one
// changelog per releasing package (writePackageChangelogs). Kept separate
// from the task closure above so it can be exercised directly in tests
// without needing a live *tr.Runner/*tr.Executor.
func (prj *Project) writeChangelogs(gomodules []*gomodule.Package) tr.Result {
	// Repos with a root module (TagPrefix == "") get one unified
	// changelog under that module's own next release tag - that's what
	// the GitHub release for the whole repo is cut against. Repos with
	// no root module (e.g. a flat addons/tools repo with only nested
	// packages, nothing tagged at the root) have no single release to
	// attach a unified changelog to, so each releasing package gets its
	// own changelog instead.
	var rootPkg *gomodule.Package
	for _, pkg := range gomodules {
		if pkg.TagPrefix == "" {
			rootPkg = pkg
			break
		}
	}
	if rootPkg == nil {
		return prj.writePackageChangelogs(gomodules)
	}
	return prj.writeRootChangelog(gomodules, rootPkg)
}

// writeRootChangelog composes every releasing package's changes into one
// changelog.Document - one section per package, titled "<import>@<tag>" -
// and writes it, rendered as Markdown, under the root package's own next
// release tag. That's the tag the GitHub release for the whole repo is cut
// against, so this is the one changelog that release's notes link to.
func (prj *Project) writeRootChangelog(gomodules []*gomodule.Package, rootPkg *gomodule.Package) tr.Result {
	doc := changelog.NewDocument("Changelog")
	for _, pkg := range gomodules {
		if !pkg.NeedsRelease || pkg.Changelog == nil {
			continue
		}
		doc.AddSection(fmt.Sprintf("%s@%s", pkg.Import, pkg.NextReleaseTag), pkg.Changelog)
	}

	md, err := doc.Render(changelog.FormatMarkdown)
	if err != nil {
		return tr.Failure(err.Error())
	}

	versionDir := filepath.Join(prj.Dist(), rootPkg.NextReleaseTag)
	if err := os.MkdirAll(versionDir, 0750); err != nil {
		return tr.Failure(err.Error())
	}
	clFilePath := filepath.Join(versionDir, "CHANGELOG.md")
	if err := os.WriteFile(clFilePath, md, 0644); err != nil {
		return tr.Failure(err.Error())
	}

	return tr.Success("changelog saved").WithDesc(clFilePath)
}

// writePackageChangelogs is releaseChangelog's fallback for repos with no
// root module: instead of one changelog attached to a whole-repo release,
// each releasing package gets its own single-section changelog.Document
// under its own next release tag (which, for a subpackage, already includes
// its own tag prefix, e.g. "daemon/v1.2.3" - joining that directly under the
// build dir naturally avoids collisions between differently-prefixed
// packages that happen to land on the same version number).
func (prj *Project) writePackageChangelogs(gomodules []*gomodule.Package) tr.Result {
	var written []string

	for _, pkg := range gomodules {
		if !pkg.NeedsRelease || pkg.Changelog == nil {
			continue
		}

		doc := changelog.NewDocument("Changelog").
			AddSection(fmt.Sprintf("%s@%s", pkg.Import, pkg.NextReleaseTag), pkg.Changelog)

		md, err := doc.Render(changelog.FormatMarkdown)
		if err != nil {
			return tr.Failure(err.Error())
		}

		versionDir := filepath.Join(prj.Dist(), pkg.NextReleaseTag)
		if err := os.MkdirAll(versionDir, 0750); err != nil {
			return tr.Failure(err.Error())
		}
		clFilePath := filepath.Join(versionDir, "CHANGELOG.md")
		if err := os.WriteFile(clFilePath, md, 0644); err != nil {
			return tr.Failure(err.Error())
		}
		written = append(written, clFilePath)
	}

	if len(written) == 0 {
		return tr.Skip("no changelogs to write")
	}
	return tr.Success(fmt.Sprintf("%d package changelog(s) saved", len(written))).WithDesc(strings.Join(written, ", "))
}
