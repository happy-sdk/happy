// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"log/slog"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/internal/cmd/hap/addons/releaser"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/sdk/cli/commands"
	"github.com/happy-sdk/happy/sdk/instance"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/settings"
)

type Settings struct {
	// happy develpment
	Happy struct {
		Placeholder settings.String `key:"placeholder"`
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
		MainArgcMax:    5,
		Description:    "Happy Prototyper provides commands to work with Happy-SDK prototypes.",
		// ThrottleTicks:  settings.Duration(time.Millisecond * 100),
		Instance: instance.Settings{
			Max: 100,
		},
	}

	config := Settings{}
	config.Happy.Placeholder = "happy-placeholder"
	config.Placeholder = "app-placeholder"

	settings.Extend(config)

	main := happy.New(settings).
		WithAddon(releaser.Addon(
			releaser.Settings{
				CommandEnabled: true,
			},
		)).
		// WithAddon(github.Addon(
		// 	github.Settings{
		// 		Owner: "happy-sdk",
		// 		Repo:  "happy",
		// 	},
		// )).
		// WithMigrations(migrations.New()).
		WithService(service()).
		WithCommand(commands.Config()).
		// WithFlag(nil).
		WithLogger(logging.Console(logging.LevelOk)).
		WithOptions(happy.Option("happy", "true"))

	return main
}

func main() {
	main := hap()

	main.BeforeAlways(func(sess *happy.Session, flags happy.Flags) error {

		loader := sess.ServiceLoader(
			"background",
		)

		<-loader.Load()

		sess.Log().Debug("SETTINGS")
		for _, s := range sess.Profile().All() {
			sess.Log().Debug(s.Key(), slog.String("value", s.Value().String()))
		}
		sess.Log().Debug("OPTIONS")
		sess.Opts().Range(func(v vars.Variable) bool {
			sess.Log().Debug(v.Name(), slog.String("value", v.String()))
			return true
		})
		sess.Log().Debug("CONFIG")
		sess.Config().Range(func(v vars.Variable) bool {
			sess.Log().Debug(v.Name(), slog.String("value", v.String()))
			return true
		})
		return loader.Err()
	})

	// main.Before(func(sess *happy.Session, args happy.Args) error {
	// 	sess.Log().Info("main.Before")
	// 	return nil
	// })

	// main.Do(func(sess *happy.Session, args happy.Args) error {
	// 	return nil
	// })

	// main.AfterSuccess(func(sess *happy.Session) error {
	// 	sess.Log().NotImplemented("main.AfterSuccess")
	// 	return nil
	// })

	// main.AfterFailure(func(sess *happy.Session, err error) error {
	// 	sess.Log().NotImplemented("main.AfterFailure")
	// 	return nil
	// })

	// main.AfterAlways(func(sess *happy.Session, err error) error {
	// 	sess.Log().NotImplemented("main.AfterAlways")
	// 	return nil
	// })

	main.Run()
}
