package main

import (
	"github.com/mkungla/happy"
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
	app := happy.New()
	app.Do(func(sess *happy.Session, args happy.Args) error {
		// name := args.Arg(0) // returns empty value if not present
		// name, err := args.ArgDefault(0, "anonymous") // return vars.Value with default value "anonymous"
		// Following is returning vars.Variable (vars.Variable can be directly logged)
		name, err := args.ArgVarDefault(0, "name", "anonymous")
		if err != nil {
			return err
		}
		sess.Log().Out("hello", name)
		return nil
	})
	app.Main()
}
