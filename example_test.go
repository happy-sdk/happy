// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy_test

import (
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/logging"
	"github.com/happy-sdk/happy/sdk/session"
)

func ExampleNew() {
	app := happy.New(happy.Settings{})

	// Create a new test logger
	log := logging.NewTestLogger(logging.LevelError)
	app.WithLogger(log)

	app.Do(func(sess *session.Context, args action.Args) error {
		sess.Log().Println("Hello, world!")
		return nil
	})

	app.Run()
	fmt.Println(log.Output())
	// Output:
	// {"level":"out","msg":"Hello, world!"}
}
