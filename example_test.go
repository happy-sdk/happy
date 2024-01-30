// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy_test

import (
	"github.com/happy-sdk/happy"
)

func ExampleNew() {
	app := happy.New(happy.Settings{})

	app.Do(func(sess *happy.Session, args happy.Args) error {
		sess.Log().Println("Hello, world!")
		return nil
	})
	// app.Run()

	// Output:
}
