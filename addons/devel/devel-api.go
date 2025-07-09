// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

import (
	"fmt"
	"sync"

	"github.com/happy-sdk/happy/sdk/api"
	"github.com/happy-sdk/happy/sdk/session"
)

type API struct {
	api.Provider
	mu  sync.RWMutex
	prj *Project
}

func NewAPI() *API {
	return &API{}
}

func (api *API) Open(sess *session.Context, prjdir string) (*Project, error) {
	fmt.Println("Opening project:", prjdir)
	return nil, nil
}
