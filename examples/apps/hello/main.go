// SPDX-License-Identifier: Apache-2.0
//
// Example application using a simple Hello World addon.
package main

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/examples/addons/hello"
)

func main() {
	app := happy.New(&happy.Settings{
		CLI: happy.CliSettings{
			WithGlobalFlags: true,
		},
	})
	app.WithAddons(hello.Addon())
	app.Run()
}
