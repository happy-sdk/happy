// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package main

import (
	"log/slog"

	"github.com/happy-sdk/happy"
)

func main() {
	app := happy.New(
		happy.Settings{
			Description: "A simple hello application",
			MainArgcMax: 1,
		},
	)

	app.Help(`Example:
  - go run ./examples/hello/
  - go run ./examples/hello/ nickname

  Increase verbosity
  - go run ./examples/hello/ --debug
  - go run ./examples/hello/ --system-debug

  Help
  - go run ./examples/hello/ -h
  `)
	app.Do(func(sess *happy.Session, args happy.Args) error {
		// name := args.Arg(0) // returns empty value if not present
		// name, err := args.ArgDefault(0, "anonymous") // return vars.Value with default value "anonymous"
		// Following is returning vars.Variable (vars.Variable can be directly logged)
		name, err := args.ArgVarDefault(0, "name", "anonymous")
		if err != nil {
			return err
		}

		sess.Log().Msg("info", slog.String("name", name.String()))

		return nil
	})
	app.Main()
}
