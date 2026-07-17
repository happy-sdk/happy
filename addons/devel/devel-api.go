// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

import (
	"sync"

	"github.com/happy-sdk/happy/addons/devel/projects"
	"github.com/happy-sdk/happy/sdk/api"
)

type API struct {
	api.Provider
	mu       sync.RWMutex
	projects *projects.API
}

func NewAPI() *API {
	return &API{
		projects: projects.New(),
	}
}

func (api *API) Projects() *projects.API {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return api.projects
}
