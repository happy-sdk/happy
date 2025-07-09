// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"sync"

	"github.com/happy-sdk/happy/cmd/gohappy/addons/projects/project"
	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/session"
)

type API struct {
	mu sync.RWMutex
	api.Provider
	prj *project.Project
}

func NewAPI() *API {
	return &API{}
}

func (api *API) Project(sess *session.Context, load bool) (*project.Project, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	if api.prj != nil {
		if load {
			if err := api.prj.Load(sess); err != nil {
				return nil, err
			}
		}
		return api.prj, nil
	}

	wd := sess.Get("app.fs.path.wd").String()
	var err error
	api.prj, err = project.Open(sess, wd)

	if err != nil {
		return nil, err
	}

	if load {
		if err := api.prj.Load(sess); err != nil {
			return nil, err
		}
	}

	return api.prj, err
}
