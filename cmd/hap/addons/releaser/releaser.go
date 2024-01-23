// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package releaser

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/settings"
)

type Settings struct {
	CommandEnabled settings.Bool `key:"command.enabled" default:"false"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func Addon(s Settings) *happy.Addon {
	addon := happy.NewAddon("releaser", s)

	return addon
}
