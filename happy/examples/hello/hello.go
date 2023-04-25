package main

import (
	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/logging"
	"golang.org/x/exp/slog"
)

// A simple hello application
//
// Usage:
// - go run ./examples/hello/
// - go run ./examples/hello/ nickname
//
// Increase verbosity
// - go run ./examples/hello/ --debug
// - go run ./examples/hello/ --system-debug
//
// Help
// - go run ./examples/hello/ -h
func main() {
	loggerConf := logging.Config{}
	logger, _ := logging.New(loggerConf)

	app := happy.NewWithLogger(logger, logging.LevelDebug)

	// func Clone[S ~[]E, E any](s S) S {

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
