// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/addons/third-party/github"
	"github.com/happy-sdk/happy/cmd/hap/addons/releaser"
	"github.com/happy-sdk/happy/cmd/hap/migrations"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/settings"
)

type Settings struct {
	// happy develpment
	Happy struct {
		Placeholder settings.Bool `key:"placeholder"`
	} `group:"happy"`
	Placeholder settings.String `key:"placeholder"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	b, err := settings.New(s)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func hap() *happy.Main {
	settings := happy.Settings{
		Name:           "Happy Prototyper",
		Slug:           "happy-sdk",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2019,
		License:        "Apache-2.0",
		TimeLocation:   "Local",
		MainArgcMax:    0,
		// ThrottleTicks:  settings.Duration(time.Millisecond * 100),
		Instance: instance.Settings{
			Max: 100,
		},
	}

	config := Settings{}
	config.Placeholder = "happy-placeholder"
	config.Placeholder = "app-placeholder"

	settings.Extend(config)

	main := happy.New(settings).
		WithAddon(releaser.Addon(
			releaser.Settings{
				CommandEnabled: true,
			},
		)).
		WithAddon(github.Addon(
			github.Settings{
				Owner: "happy-sdk",
				Repo:  "happy",
			},
		)).
		WithMigrations(migrations.New()).
		WithService(service()).
		// WithCommand(nil).
		// WithFlag(nil).
		WithLogger(logging.Console(logging.LevelSystemDebug)).
		WithOptions(happy.Option("somekey", "somevalue"))

	return main
}

func githubAddon() *happy.Addon {
	return github.Addon(
		github.Settings{
			Owner: "happy-sdk",
			Repo:  "happy",
		},
	)
}
