// SPDX-License-Identifier: Apache-2.0
//
// A minimal Hello World addon example.
package hello

import (
	"fmt"

	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/addon"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

// Addon returns a simple addon that registers a single "hello" command.
func Addon() *addon.Addon {
	ad := addon.New("Hello")

	// Provide a simple "hello" command.
	helloCmd := command.New("hello",
		command.Config{
			Description: "Prints Hello from addon",
			MaxArgs:     1,
		})

	helloCmd.Do(func(sess *session.Context, args action.Args) error {
		greet, err := args.ArgDefault(0, "World")
		if err != nil {
			return err
		}
		fmt.Printf("Hello, %s!\n", greet)
		return nil
	})

	ad.ProvideCommands(helloCmd)

	return ad
}
