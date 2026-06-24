// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package happy_test

import (
	"bytes"
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/logging"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/session"
)

func ExampleNew() {
	app := happy.New(nil)

	// Route logs to an in-memory buffer with a deterministic, timestamp-free
	// JSON format so this example's output is reproducible.
	buf := new(bytes.Buffer)
	cnf := logging.DefaultConfig()
	cnf.NoTimestamp = true
	app.WithLogger(logging.New(cnf, logging.NewJSONAdapter(buf)))

	app.Do(func(sess *session.Context, args action.Args) error {
		sess.Log().Out("Hello, world!")
		// app.Run() below exits the process once this action returns, so the
		// buffer must be printed here rather than after Run() returns.
		fmt.Print(buf.String())
		return nil
	})

	app.Run()
	// Output:
	// {"level":"out","msg":"Hello, world!"}
}
