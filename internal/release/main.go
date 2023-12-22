// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"errors"
	"log/slog"
	"os/exec"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy-go/internal/release/releaser"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/sdk/cli"
	"github.com/happy-sdk/happy/sdk/commands"
	"golang.org/x/text/language"
)

type Settings struct {
	happy.Settings
	GithubToken  settings.String `key:"github.token" mutation:"once"`
	MinGoVersion settings.String `key:"go.version.min" mutation:"once"`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	blueprint, err := settings.NewBlueprint(s)
	if err != nil {
		return nil, err
	}
	blueprint.Describe("github.token", language.English, "Github token")
	return blueprint, nil
}

func main() {
	settings := happy.Settings{
		Name:           "Happy-Go - releaser",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2023,
		License:        "Apache License 2.0",
		Logger: logging.Settings{
			Secrets: "token",
		},
	}
	settings.Extend("happy-go", Settings{
		GithubToken:  "",
		MinGoVersion: "1.21.5",
	})

	app := happy.New(settings)

	// Happy CLI commands
	app.AddCommand(commands.Info())
	app.AddCommand(commands.Reset())

	var release *releaser.Releaser

	app.Before(func(sess *happy.Session, args happy.Args) error {
		if !sess.Profile().Get("happy-go.github.token").IsSet() {
			return nil // not retruning error here so that we can call other subcommands
		}

		gitstatus := exec.Command("git", "diff-index", "--quiet", "HEAD")
		if err := cli.RunCommand(sess, gitstatus); err != nil {
			return errors.New("git is in a dirty state")
		}
		release = releaser.New(sess.Get("app.fs.path.pwd").String(), []string{
			"internal/release",
		})
		return release.Before(sess, args)
	})

	app.Do(func(sess *happy.Session, args happy.Args) error {
		if !sess.Profile().Get("happy-go.github.token").IsSet() {
			return errors.New("github.token is not set")
		}
		sess.Log().Info("using GITHUB_TOKEN", slog.String("token", sess.Profile().Get("happy-go.github.token").String()))
		sess.Log().Info("do")
		return release.Do(sess, args)
	})
	app.Main()
}
