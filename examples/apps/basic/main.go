// SPDX-License-Identifier: Apache-2.0
//
// Minimal Happy application example using built-in CLI features.
package main

import (
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/session"
)

func main() {
	// In a real application, you would usually load settings from a file or environment.
	// Here we construct them inline and list most sections, with defaults left commented
	// so it is easy to discover what is available.
	app := happy.New(&happy.Settings{
		// Application name
		Name:        "Basic",
		Description: "Basic example using Happy-SDK builtins",
		Slug:        "basic",
		// Identifier:     "com.github.happy-sdk.happy.examples.apps.basic",
		// CopyrightBy:    "Anonymous",
		// CopyrightSince: 2025,
		// License:        "NOASSERTION",

		// Engine: happy.EngineSettings{},
		CLI: happy.CliSettings{
			// Enable built-in CLI features:
			WithConfigCmd:   true, // adds `config` command
			WithI18nCmd:     true, // adds `i18n` command
			WithGlobalFlags: true, // adds standard global flags (help, version, verbose, etc.)
			// HideDisabledCommands: false,
			MainMinArgs: 0,
			MainMaxArgs: 1,
		},
		// Profiles: happy.ProfileSettings{},
		// DateTime: happy.DateTimeSettings{},
		Instance: happy.InstanceSettings{
			// Max: 0,
		},
		Logging: happy.LoggingSettings{
			// Set default above info, so --verbose has a visible effect.
			Level: logging.LevelSuccess,
			// WithSource:      false,
			// TimestampFormat: "",
			// NoTimestamp:     false,
			// NoSlogDefault:   false,
		},
		// Services: happy.ServicesSettings{},
		// Stats:    happy.StatsSettings{},
		// Devel:    happy.DevelSettings{},
		// I18n:     happy.I18nSettings{},
	})

	app.Do(func(sess *session.Context, args action.Args) error {
		greet, err := args.ArgDefault(0, "World")
		if err != nil {
			return err
		}
		fmt.Printf("Hello, %s!\n", greet)
		return nil
	})

	app.Run()
}
