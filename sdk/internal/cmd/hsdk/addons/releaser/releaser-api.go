// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/happy-sdk/happy/sdk/app/session"
	"github.com/happy-sdk/happy/sdk/custom"
	"github.com/happy-sdk/happy/sdk/internal/cmd/hsdk/addons/releaser/module"
	"golang.org/x/mod/semver"
)

type releaser struct {
	custom.API
	mu       sync.RWMutex
	sess     *session.Context
	packages []*module.Package
	queue    []string
}

func newReleaser() *releaser {
	return &releaser{}
}

func (r *releaser) Initialize(sess *session.Context, path string, allowDirty bool) error {
	if err := newConfiguration(sess, path, allowDirty); err != nil {
		return err
	}
	r.mu.Lock()
	r.sess = sess
	r.mu.Unlock()
	sess.Log().Ok("releaser initialized", slog.String("wd", sess.Get("releaser.wd").String()))
	return nil
}

func (r *releaser) Run(next string) error {
	if err := r.confirmConfig(next); err != nil {
		return err
	}

	if err := r.loadModules(); err != nil {
		return err
	}

	if err := r.releaseModules(); err != nil {
		return err
	}
	r.sess.Log().Ok("releaser done")

	return r.printChangelog()
}

func (r *releaser) releaseModules() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sess.Log().Info("releasing modules")

	for _, q := range r.queue {
		for _, pkg := range r.packages {
			if pkg.Import == q {
				if err := pkg.Release(r.sess); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *releaser) confirmConfig(next string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.sess.Opts().Set("releaser.next", next); err != nil {
		return err
	}
	if r.sess.Get("releaser.go.modules.count").Int() == 0 {
		return fmt.Errorf("no modules to release")
	}
	m, err := getConfirmConfigModel(r.sess)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	var userSelectedYes bool
	if model, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	} else {
		m, ok := model.(configTable)
		if !ok {
			return errors.New("could not assert model type")
		}
		userSelectedYes = m.yes
	}
	if !userSelectedYes {
		return errors.New("release canceled by user")
	}
	return nil
}

func (r *releaser) loadModules() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sess.Log().Info("loading modules")

	var pkgs []*module.Package
	if err := filepath.Walk(r.sess.Get("releaser.wd").String(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		goModPath := filepath.Join(path, "go.mod")
		if _, err := os.Stat(goModPath); err != nil {
			return nil
		}
		fmt.Println(goModPath)
		pkg, err := module.Load(goModPath)
		if err != nil {
			return err
		}
		pkgs = append(pkgs, pkg)
		return nil
	}); err != nil {
		return err
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("no modules found in %s", r.sess.Get("releaser.wd").String())
	}

	for _, pkg := range pkgs {
		r.sess.Log().Info("loading release info for", slog.String("pkg", pkg.Modfile.Module.Mod.Path))
		tagPrefix := strings.TrimPrefix(pkg.Dir+"/", r.sess.Get("releaser.wd").String()+"/")
		pkg.TagPrefix = tagPrefix

		if err := pkg.LoadReleaseInfo(r.sess); err != nil {
			return err
		}
	}

	commonDeps, err := module.GetCommonDeps(pkgs)
	if err != nil {
		return err
	}
	for _, dep := range commonDeps {
		if semver.Compare(dep.MinVersion, dep.MaxVersion) != 0 {
			r.sess.Log().Info("common dep",
				slog.String("dep", dep.Import),
				slog.String("version.max", dep.MaxVersion),
				slog.String("version.min", dep.MinVersion),
				slog.Int("used.by", len(dep.UsedBy)),
			)
			for _, imprt := range dep.UsedBy {
				for _, pkg := range pkgs {
					if pkg.Import == imprt {
						r.sess.Log().Info("update dep",
							slog.String("pkg", pkg.Import),
							slog.String("dep", dep.Import),
							slog.String("dep.version", dep.MaxVersion),
						)
						if err := pkg.SetDep(dep.Import, dep.MaxVersion); err != nil {
							return err
						}
						break
					}
				}
			}
		}
	}

	queue, err := module.TopologicalReleaseQueue(pkgs)
	if err != nil {
		return err
	}
	for _, p := range queue {
		r.sess.Log().Info("release queue", slog.String("pkg", p))
	}

	m, err := module.GetConfirmReleasablesView(r.sess, pkgs, queue)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	var userSelectedYes bool
	if model, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	} else {
		m, ok := model.(module.ReleasablesTableView)
		if !ok {
			return errors.New("could not assert model type")
		}
		userSelectedYes = m.Yes
	}
	if !userSelectedYes {
		return errors.New("release canceled by user")
	}

	r.queue = queue
	r.packages = pkgs
	return nil
}

type fullChangelog struct {
	Root    *packageChangelog
	Subpkgs []*packageChangelog
}

type packageChangelog struct {
	pkg      *module.Package
	Breaking []string
	Changes  []string
}

func (r *releaser) printChangelog() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cl := &fullChangelog{}

	for _, pkg := range r.packages {
		if !pkg.NeedsRelease || (pkg.Changelog == nil || pkg.Changelog.Empty()) {
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

		if pkg.Dir == r.sess.Get("releaser.wd").String() {
			cl.Root = clp
		} else {
			cl.Subpkgs = append(cl.Subpkgs, clp)
		}
	}

	fmt.Println("## Changelog")

	fmt.Printf("`%s@%s`\n\n", cl.Root.pkg.Import, cl.Root.pkg.NextRelease)

	if cl.Root == nil {
		return nil
	}
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
		fmt.Println("### Breaking Changes")
		fmt.Println(breakingsection)
	}

	var changessection string
	for _, change := range cl.Root.Changes {
		found := false
		for _, scl := range cl.Subpkgs {
			for _, bcl := range scl.Changes {
				if bcl == change {
					found = true
					break
				}
			}
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
		fmt.Println("### Changes")
		fmt.Println(changessection)
	}
	fmt.Println("")

	for _, scl := range cl.Subpkgs {
		fmt.Printf("### %s\n\n`%s@%s`\n", scl.pkg.NextRelease, scl.pkg.Import, scl.pkg.NextRelease)

		for i, breaking := range scl.Breaking {
			if i == 0 {
				fmt.Println("**Breaking Changes**")
			}
			fmt.Println(breaking)
		}
		for i, change := range scl.Changes {
			if i == 0 {
				fmt.Println("**Changes**")
			}
			fmt.Println(change)
		}
	}

	fmt.Println("")
	return nil
}
