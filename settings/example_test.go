// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors

package settings_test

// type Settings struct {
// 	Username settings.String `default:"anonymous" mutation:"once"`
// }

// func (s Settings) Blueprint() (*settings.Blueprint, error) {
// 	return settings.New(s)
// }

// func handleErr(err error) {
// 	if err == nil {
// 		return
// 	}
// 	panic(fmt.Sprintf("ERROR: %s", err))
// }

// func ExampleNew() {
// 	// Create main application settings blueprint
// 	blueprint, err := settings.New(happy.Settings{
// 		Name: "Happy",
// 	})
// 	handleErr(err)
// 	blueprint.Extend("happy-go", Settings{})

// 	// Compile a schema for current app version.
// 	schema, err := blueprint.Schema("github.com/happy-sdk/happy/pkg/settings", "1.0.0")
// 	handleErr(err)

// 	profile, err := schema.Profile("default", nil)
// 	handleErr(err)

// 	for _, setting := range profile.All() {
// 		fmt.Printf("%s=%q\n", setting.Key(), setting.String())
// 	}

// 	// OUTPUT:
// 	// name="Happy Prototype"
// 	// happy-go.username="anonymous"
// }
