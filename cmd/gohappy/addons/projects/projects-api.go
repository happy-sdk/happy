// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"fmt"
	"path"
	"sync"

	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects/project"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/session"
)

type API struct {
	mu sync.RWMutex
	api.Provider
	currentPrj *project.Project
}

func NewAPI() *API {
	return &API{}
}

func (api *API) OpenProject(sess *session.Context, wd string, load bool) (prj *project.Project, err error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Project can be loaded from same wd not changing projects
	if api.currentPrj != nil {
		if api.currentPrj.WD() != wd {
			return nil, fmt.Errorf(
				"%w: can not load project from %s already detected or loaded at %s",
				Error, wd, api.currentPrj.WD())
		}

		if load && !api.currentPrj.Detected() && !api.currentPrj.Loaded() {
			if err := api.currentPrj.Load(sess); err != nil {
				return nil, err
			}
		}
		return api.currentPrj, nil
	}

	// Attempt to detect project from wd
	prj, err = project.Open(sess, wd)
	if err != nil {
		return
	}
	api.currentPrj = prj
	if load {
		err = prj.Load(sess)
	}
	return
}

func (a *API) Project() (*project.Project, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.currentPrj == nil {
		return nil, fmt.Errorf("%w: no project", project.Error)
	}
	return a.currentPrj, nil
}

func (a *API) Detect(sess *session.Context) bool {
	wd := sess.Get("app.fs.path.wd").String()
	prj, err := a.OpenProject(sess, wd, false)
	if err != nil {
		sess.Log().Error(err.Error())
		return false
	}
	return prj.Detected()
}

func (a *API) ProjectInfoPrint(sess *session.Context) error {
	prj, err := a.Project()
	if err != nil {
		return err
	}

	info := textfmt.Table{
		Title:      "Project Information",
		WithHeader: true,
	}
	info.AddRow("key", "value")
	prj.Config().Range(func(opt options.Option) bool {
		info.AddRow(opt.Key(), opt.String())
		return true
	})
	fmt.Println(info.String())

	modules, err := prj.GoModules(sess, true)
	if err == nil {
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
