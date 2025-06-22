// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"errors"

	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/addon"
)

var (
	Error = errors.New("project")
)

type Settings struct {
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(&s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func Addon() *addon.Addon {
	addon := addon.New("Projects").
		WithConfig(addon.Config{}).
		WithSettings(Settings{}).
		WithOptions(
			addon.Option("next", "auto", "specify next version to release auto|major|minor|patch", false, nil),
		).WithAPI(NewAPI())

	return addon
}
