// SPDX-License-Identifier: Apache-2.0
//
// Example application enabling built-in global CLI flags.
package main

import (
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/session"
)

func main() {
	app := happy.New(nil)

	app.Do(func(sess *session.Context, args action.Args) error {
		fmt.Println("Hello, happy!")
		return nil
	})

	app.Run()
}
