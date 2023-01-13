package main

import (
	"github.com/mkungla/happy"
)

// A simple hello application
//
// Usage:
// - go run ./examples/hello/
// - go run ./examples/hello/ nickname
// - go run ./examples/hello/ -h
func main() {
	app := happy.New()
	app.Do(func(sess *happy.Session, args happy.Args) error {
		name, err := args.ArgVarDefault(0, "name", "anonymous")
		if err != nil {
			return err
		}
		sess.Log().Out("hello", name)
		return nil
	})
	app.Main()
}
