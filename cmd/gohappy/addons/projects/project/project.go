// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/happy-sdk/happy/cmd/gohappy/pkg/git"
	"github.com/happy-sdk/happy/cmd/gohappy/pkg/gomodule"
	"github.com/happy-sdk/happy/pkg/fsutils"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/session"
)

var Error = errors.New("project")

type Project struct {
	mu       sync.RWMutex
	config   *options.Options
	root     string
	detected bool
	loaded   bool

	gomodules []*gomodule.Package
}

func Open(sess *session.Context, dir string) (*Project, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: empty path", Error)
	}

	if !fsutils.IsDir(dir) {
		return nil, fmt.Errorf("%w: not a project directory", Error)
	}

	config, err := newConfig()
	if err != nil {
		return nil, err
	}

	if err := config.Set("local.wd", dir); err != nil {
		return nil, err
	}

	if err := git.DetectGitRepo(sess, config); err != nil {
		return nil, err
	}

	prj := &Project{
		config: config,
	}

	if config.Get("git.repo.found").Variable().Bool() {
		prj.detected = true
		prj.root = config.Get("git.repo.root").String()
	} else {
		prj.root = dir
	}

	// nil, fmt.Errorf("%w: no project found in %s", project.Error, wd)
	return prj, nil
}

func (p *Project) Load(sess *session.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.loaded {
		return nil
	}

	// Git info
	if p.config.Get("git.repo.found").Variable().Bool() {
		if err := git.LoadInfo(sess, p.config); err != nil {
			return err
		}
	}

	// Go modules
	sess.Log().Debug("count modules")
	moduleCount, err := gomodule.Count(p.root)
	if err != nil {
		return err
	}

	if err := p.config.Set("go.module.count", moduleCount); err != nil {
		return err
	}

	if err := p.config.Set("go.monorepo", moduleCount > 1); err != nil {
		return err
	}

	// env
	dotenvp := filepath.Join(p.root, ".env")
	dotenvb, err := os.ReadFile(dotenvp)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: failed to open env file: %s", Error, err.Error())
	}
	sess.Log().Debug("loading .env file", slog.String("path", dotenvp))

	env, err := vars.ParseMapFromBytes(dotenvb)
	if err != nil {
		return err
	}

	env.Range(func(v vars.Variable) bool {
		if err = p.config.Set(fmt.Sprintf("env.%s", v.Name()), v.String()); err != nil {
			sess.Log().Error("error loading env var", slog.String("env", v.Name()), slog.String("value", v.String()), slog.String("err", err.Error()))
		}
		return true
	})

	return nil
}

func (p *Project) Config() (config *options.Options) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	config = p.config
	return
}

func (p *Project) GoModules(sess *session.Context, fresh, checkRemote bool) ([]*gomodule.Package, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config.Get("go.module.count").Variable().Int() == 0 {
		return nil, fmt.Errorf("%w: no go modules", Error)
	}

	if !fresh && p.gomodules != nil {
		return p.gomodules, nil
	}

	var (
		gomodules []*gomodule.Package
		err       error
	)

	sess.Log().Debug("loading modules")
	if gomodules, err = gomodule.LoadAll(sess, p.root); err != nil {
		return nil, err
	}

	for _, pkg := range gomodules {

		if err := pkg.LoadReleaseInfo(sess, p.root, p.config.Get("git.repo.remote.name").String(), checkRemote); err != nil {
			return nil, err
		}
	}

	if len(gomodules) > 0 {
		if _, err = gomodule.TopologicalReleaseQueue(gomodules); err != nil {
			return nil, err
		}
		p.gomodules = gomodules
	}

	return p.gomodules, nil
}

func (p *Project) UpdateTopologicalReleaseQueue() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	gomodules := p.gomodules
	if _, err := gomodule.TopologicalReleaseQueue(gomodules); err != nil {
		return err
	}
	p.gomodules = gomodules
	return nil
}

func (p *Project) WD() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	wd := p.config.Get("git.repo.root").String()
	return wd
}

func (p *Project) Dirty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config.Get("git.repo.dirty").Variable().Bool()
}
