// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"fmt"
	"path"
	"sync"

	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects/project"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/session"
)

type API struct {
	mu sync.RWMutex
	api.Provider
	loaded bool
	wd     string
	prj    *project.Project
}

func NewAPI() *API {
	return &API{}
}

func (a *API) Load(sess *session.Context, wd string) (err error) {
	if a.isLoaded() {
		a.mu.RLock()
		defer a.mu.RUnlock()
		return fmt.Errorf("%w: project already loaded at %s", Error, a.wd)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.prj, err = project.Load(sess, wd)
	if err != nil {
		return err
	}

	a.loaded = true
	a.wd = wd
	return nil
}

func (a *API) Project() (*project.Project, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.prj == nil {
		return nil, fmt.Errorf("%w: project not loaded", Error)
	}
	return a.prj, nil
}

func (a *API) PrintInfo() error {
	prj, err := a.Project()
	if err != nil {
		return err
	}

	info := textfmt.Table{
		Title: "Project Information",
	}
	for _, entry := range prj.Config().All() {
		info.AddRow(entry.Name(), entry.String())
	}
	fmt.Println(info.String())

	modules := prj.GoModules()
	if modules != nil {
		modulelist := textfmt.Table{
			Title: "Packages",
		}
		modulelist.AddRow(
			"Package",
			"Action",
			"Current",
			"Next",
			"Update deps",
		)
		for _, pkg := range modules {
			action := "skip"
			if pkg.NeedsRelease {
				action = "release"
			}
			if pkg.FirstRelease {
				action = "initial"
			}
			modulelist.AddRow(
				pkg.Import,
				action,
				path.Base(pkg.LastRelease),
				path.Base(pkg.NextRelease),
				fmt.Sprint(pkg.UpdateDeps),
			)
		}
		fmt.Println(modulelist.String())
	}
	return nil
}

func (a *API) isLoaded() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.loaded
}
