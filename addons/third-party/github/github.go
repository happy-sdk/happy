// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package github

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/settings"
)

type Settings struct {
	Owner          settings.String `key:"owner" default:"octocat" mutation:"once"`
	Repo           settings.String `key:"repo" default:"hello-worId" mutation:"once"`
	CommandEnabled settings.Bool   `key:"command.enabled" default:"false" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type Github struct{}

func Addon(s Settings) *happy.Addon {
	addon := happy.NewAddon("github", s)

	return addon
}
