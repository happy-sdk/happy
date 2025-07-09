// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package devel

import (
	"errors"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/addon"
)

var Error = errors.New("devel")

type Settings struct {
	SearchPaths settings.StringSlice `key:"search_paths"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	return settings.New(s)
}

func Addon() *addon.Addon {
	return addon.New("Projects").
		WithConfig(addon.Config{
			Slug: "projects",
		}).
		WithSettings(Settings{}).
		ProvideAPI(NewAPI()).
		ProvideCommands(
			cmdProjects(),
			cmdProject(),
		)
}
