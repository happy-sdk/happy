// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

import (
	"errors"

	"github.com/happy-sdk/happy/addons/devel/projects"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/addon"
	"github.com/happy-sdk/happy/sdk/session"
)

var (
	Error = errors.New("devel")
)

type Settings struct {
	// Projects holds the project discovery configuration (search paths and
	// ignore patterns). It's not read directly in this package - WithSettings
	// below registers these values as the "devel.projects.*" runtime options,
	// which the projects API reads via sess.Get(...) when listing projects.
	Projects projects.Settings `key:"projects"`
}

func (s *Settings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

func Addon(s Settings) *addon.Addon {
	api := NewAPI()
	return addon.New("Devel").
		WithConfig(addon.Config{
			Slug: "devel",
		}).
		WithSettings(&s).
		ProvideAPI(api).
		OnRegister(func(sess session.Register) error {
			return nil
		})
}
