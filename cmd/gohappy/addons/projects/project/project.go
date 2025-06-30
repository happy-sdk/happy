// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package project

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/happy-sdk/happy/cmd/gohappy/pkg/git"
	"github.com/happy-sdk/happy/cmd/gohappy/pkg/module"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/session"
)

var Error = errors.New("project")

type Project struct {
	mu       sync.RWMutex
	detected bool // project detected
	loaded   bool // project loaded

	config    *options.Options
	gomodules []*module.Package

	root string
}

func Open(sess *session.Context, wd string) (*Project, error) {
	config, err := newConfig()
	if err != nil {
		return nil, err
	}
	prj := &Project{
		config: config,
		root:   wd,
	}
	if err := config.Set("local.wd", wd); err != nil {
		return nil, err
	}

	if err := git.DetectGitRepo(sess, config); err != nil {
		return nil, err
	}

	if config.Get("git.repo.found").Variable().Bool() {
		prj.detected = true
		prj.root = config.Get("git.repo.root").String()
	}

	return prj, nil
}

func (p *Project) Load(sess *session.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.root == "" {
		return fmt.Errorf("project root path is not set")
	}

	if p.config.Get("git.repo.found").Variable().Bool() {
		if err := git.LoadInfo(sess, p.config); err != nil {
			return err
		}
	}

	sess.Log().Debug("count modules")
	moduleCount, err := module.Count(p.root)
	if err != nil {
		return err
	}
	if err := p.config.Set("go.module.count", moduleCount); err != nil {
		return err
	}
	if err := p.config.Set("go.monorepo", moduleCount > 1); err != nil {
		return err
	}

	dotenvp := filepath.Join(p.root, ".env")
	dotenvb, err := os.ReadFile(dotenvp)
	if err == nil {
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
	}

	return nil
}

func (p *Project) Config() (config *options.Options) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	config = p.config
	return
}

func (p *Project) Loaded() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loaded
}

func (p *Project) Detected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.detected
}

func (p *Project) WD() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	wd := p.config.Get("local.wd").String()
	return wd
}

func (p *Project) GoModules(sess *session.Context, fresh bool) ([]*module.Package, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config.Get("go.module.count").Variable().Int() == 0 {
		return nil, fmt.Errorf("%w: no go modules", Error)
	}

	if !fresh && p.gomodules != nil {
		return p.gomodules, nil
	}

	var (
		gomodules []*module.Package
		err       error
	)

	sess.Log().Debug("loading modules")
	if gomodules, err = module.LoadAll(sess, p.root); err != nil {
		return nil, err
	}

	for _, pkg := range gomodules {
		tagPrefix := strings.TrimPrefix(pkg.Dir+"/", p.root+"/")
		if err := pkg.LoadReleaseInfo(sess, p.root, tagPrefix); err != nil {
			return nil, err
		}
	}

	if _, err = module.TopologicalReleaseQueue(gomodules); err != nil {
		return nil, err
	}
	p.gomodules = gomodules

	return p.gomodules, nil
}
