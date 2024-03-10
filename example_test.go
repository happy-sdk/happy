// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy_test

import (
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/sdk/logging"
)

func ExampleNew() {
	log := logging.NewTestLogger(logging.LevelError)

	app := happy.New(happy.Settings{})
	app.WithLogger(log)
	app.Do(func(sess *happy.Session, args happy.Args) error {
		sess.Log().Println("Hello, world!")
		return nil
	})

	app.Run()
	fmt.Println(log.Output())
	// Output:
	// {"level":"out","msg":"Hello, world!"}
}
