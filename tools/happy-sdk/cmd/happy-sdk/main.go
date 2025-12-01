// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package main

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/tools/happy-sdk/internal/cmd"
)

func main() {
	app := happy.New(&happy.Settings{
		Name:        "happy-sdk",
		Slug:        "happy-sdk",
		Description: "Happy SDK maintainer tools",
		CopyrightBy: "The Happy Authors",
		License:     "Apache-2.0",
		CLI: happy.CliSettings{
			WithGlobalFlags: true,
			WithI18nCmd:     true, // Enable i18n command for maintainers
		},
		Logging: happy.LoggingSettings{
			Level: logging.LevelSuccess,
		},
	})

	// Add happy-sdk commands
	cmd.SetupCommands(app)

	app.Run()
}
