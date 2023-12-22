// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings_test

import (
	"fmt"

	"github.com/happy-sdk/happy-go/settings"
)

type Settings struct {
	Name     settings.String `default:"Happy Prototype" mutation:"once"`
	Verbose  settings.Bool   `default:"true" mutation:"mutable"`
	Username settings.String `default:"anonymous" mutation:"once" required`
}

func (s Settings) Blueprint() (*settings.Blueprint, error) {
	return settings.NewBlueprint(s)
}

func handleErr(err error) {
	if err == nil {
		return
	}
	panic(fmt.Sprintf("ERROR: %s", err))
}

func out(a ...any) {
	fmt.Println(a...)
}

func ExampleNew() {
	// Create main application settings blueprint
	blueprint, err := settings.NewBlueprint(Settings{
		Verbose: false, // Set verbose true
	})
	handleErr(err)

	// Compile a schema for current app version.
	schema, err := blueprint.Schema("github.com/happy-sdk/happy/pkg/settings", "1.0.0")
	handleErr(err)

	profile, err := schema.Profile("default", nil)
	handleErr(err)

	for _, setting := range profile.All() {
		fmt.Printf("%s=%q\n", setting.Key(), setting.String())
	}

	// OUTPUT:
	// name="Happy Prototype"
	// username="anonymous"
	// verbose="true"
}
