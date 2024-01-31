// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/internal/cmd/hap/addons/releaser/module"
	"golang.org/x/mod/semver"
)

func (r *releaser) Initialize(sess *happy.Session, path string) error {
	config, err := newConfiguration(sess, path)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.config = *config
	r.sess = sess
	r.mu.Unlock()
	sess.Log().Ok("releaser initialized", slog.String("wd", config.WD))
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
	sess, err := r.session()
	if err != nil {
		return err
	}
	sess.Log().Ok("releaser done")
	return nil
}

func (r *releaser) session() (*happy.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.sess == nil {
		return nil, fmt.Errorf("releaser not initialized with session")
	}
	return r.sess, nil
}

func (r *releaser) confirmConfig(next string) error {
	sess, err := r.session()
	if err != nil {
		return err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.sess.Set("releaser.next", next); err != nil {
		return err
	}
	if sess.Get("releaser.go.modules.count").Int() == 0 {
		return fmt.Errorf("no modules to release")
	}
	m, err := r.config.getConfirmConfigModel(sess)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	var userSelectedYes bool
	if model, err := p.Run(); err != nil {
		return fmt.Errorf("Error running program: %w", err)
	} else {
		m, ok := model.(configTable)
		if !ok {
			return fmt.Errorf("Could not assert model type.")
		}
		userSelectedYes = m.yes
	}
	if !userSelectedYes {
		return fmt.Errorf("release canceled by user.")
	}

	return nil
}

func (r *releaser) loadModules() error {
	sess, err := r.session()
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	sess.Log().Info("loading modules")

	if len(r.config.WD) < 5 {
		return fmt.Errorf("invalid working directory: %s", r.config.WD)
	}

	var pkgs []*module.Package
	if err := filepath.Walk(r.config.WD, func(path string, info os.FileInfo, err error) error {
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
		pkg, err := module.Load(goModPath)
		if err != nil {
			return err
		}
		pkgs = append(pkgs, pkg)
		return nil
	}); err != nil {
		return err
	}

	for _, pkg := range pkgs {
		sess.Log().Info("loading release info for", slog.String("pkg", pkg.Modfile.Module.Mod.Path))
		tagPrefix := strings.TrimPrefix(pkg.Dir+"/", r.config.WD+"/")
		pkg.TagPrefix = tagPrefix

		if err := pkg.LoadReleaseInfo(sess); err != nil {
			return err
		}
	}

	commonDeps, err := module.GetCommonDeps(pkgs)
	if err != nil {
		return err
	}
	for _, dep := range commonDeps {
		if semver.Compare(dep.MinVersion, dep.MaxVersion) != 0 {
			sess.Log().Info("common dep",
				slog.String("dep", dep.Import),
				slog.String("version.max", dep.MaxVersion),
				slog.String("version.min", dep.MinVersion),
				slog.Int("used.by", len(dep.UsedBy)),
			)
			for _, imprt := range dep.UsedBy {
				for _, pkg := range pkgs {
					if pkg.Import == imprt {
						sess.Log().Info("update dep",
							slog.String("pkg", pkg.Import),
							slog.String("dep", dep.Import),
							slog.String("dep.version", dep.MaxVersion),
						)
						pkg.SetDep(dep.Import, dep.MaxVersion)
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
		sess.Log().Info("release queue", slog.String("pkg", p))
	}

	m, err := module.GetConfirmReleasablesView(sess, pkgs, queue)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	var userSelectedYes bool
	if model, err := p.Run(); err != nil {
		return fmt.Errorf("Error running program: %w", err)
	} else {
		m, ok := model.(module.ReleasablesTableView)
		if !ok {
			return fmt.Errorf("Could not assert model type.")
		}
		userSelectedYes = m.Yes
	}
	if !userSelectedYes {
		return fmt.Errorf("release canceled by user.")
	}

	r.queue = queue
	r.packages = pkgs
	return nil
}

func (r *releaser) releaseModules() error {
	sess, err := r.session()
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	sess.Log().Info("releasing modules")

	for _, q := range r.queue {
		for _, pkg := range r.packages {
			if pkg.Import == q {
				if err := pkg.Release(sess); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
