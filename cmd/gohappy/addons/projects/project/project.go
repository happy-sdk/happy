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

	"github.com/happy-sdk/happy/cmd/gohappy/pkg/git"
	"github.com/happy-sdk/happy/cmd/gohappy/pkg/module"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/session"
)

var Error = errors.New("project")

type Project struct {
	config    *vars.Map
	sess      *session.Context
	gomodules []*module.Package
}

func Load(sess *session.Context, wd string) (*Project, error) {
	currentPath, err := filepath.Abs(wd)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(currentPath)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("%w: %s is not a directory", Error, currentPath)
	}
	config := vars.NewMap()

	if err := config.Store("wd", currentPath); err != nil {
		return nil, err
	}

	sess.Log().Debug("load git info")
	if err := git.LoadInfo(sess, config, currentPath); err != nil {
		return nil, err
	}

	rootPath := currentPath
	if config.Get("git.repo.found").Bool() {
		rootPath = config.Get("git.repo.root").String()
	}

	sess.Log().Debug("count modules")
	moduleCount, err := module.Count(rootPath)
	if err != nil {
		return nil, err
	}
	if err := config.Store("go.module.count", moduleCount); err != nil {
		return nil, err
	}
	if err := config.Store("go.monorepo", moduleCount > 1); err != nil {
		return nil, err
	}

	dotenvp := filepath.Join(rootPath, ".env")
	dotenvb, err := os.ReadFile(dotenvp)
	if err == nil {
		sess.Log().Debug("loading .env file", slog.String("path", dotenvp))
		env, err := vars.ParseMapFromBytes(dotenvb)
		if err != nil {
			return nil, err
		}
		env.Range(func(v vars.Variable) bool {
			if err = config.Store(fmt.Sprintf("env.%s", v.Name()), v.String()); err != nil {
				sess.Log().Error("error loading env var", slog.String("env", v.Name()), slog.String("value", v.String()), slog.String("err", err.Error()))
			}
			return true
		})
	}

	var gomodules []*module.Package
	if moduleCount > 0 {
		sess.Log().Debug("loading modules")
		if gomodules, err = module.LoadAll(sess, rootPath); err != nil {
			return nil, err
		}

		for _, pkg := range gomodules {
			tagPrefix := strings.TrimPrefix(pkg.Dir+"/", rootPath+"/")
			if err := pkg.LoadReleaseInfo(sess, rootPath, tagPrefix); err != nil {
				return nil, err
			}
		}
		_, err := module.TopologicalReleaseQueue(gomodules)
		if err != nil {
			return nil, err
		}
	}

	return &Project{
		config:    config,
		sess:      sess,
		gomodules: gomodules,
	}, nil
}

func (p *Project) Config() *vars.Map {
	return p.config
}

func (p *Project) GoModules() []*module.Package {
	return p.gomodules
}
