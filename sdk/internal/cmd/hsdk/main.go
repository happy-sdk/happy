// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package main

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/internal/cmd/hsdk/addons/releaser"
)

func main() {
	app := happy.New(happy.Settings{
		Name:           "Happy SDK CLI",
		Slug:           "happy-sdk",
		Description:    "Happy SDK Command Line Interface provides commands to work with Happy-SDK",
		License:        "Apache-2.0",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2019,
	}).WithAddon(releaser.Addon())

	app.Run()
}
