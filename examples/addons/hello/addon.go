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
			// Usage            settings.String `key:"usage" mutation:"once"`
			// HideDefaultUsage settings.Bool   `key:"hide_default_usage" default:"false"`
			// Category         settings.String `key:"category"`
			Description: "Prints Hello from addon",
			// MinArgs    settings.Uint `key:"min_args" default:"0" mutation:"once"`
			// MinArgsErr settings.String
			MaxArgs: 1,
			// MaxArgsErr settings.String
			// SharedBeforeAction settings.Bool `key:"shared_before_action" default:"false"`
			// Immediate settings.Bool `key:"immediate" default:"false"`
			// SkipSharedBefore settings.Bool `key:"skip_shared_before" default:"false"`
			// Disabled settings.Bool `key:"disabled" default:"false" mutation:"mutable"`
			// FailDisabled settings.Bool `key:"fail_disabled" default:"false"`
		})

	helloCmd.Do(func(sess *session.Context, args action.Args) error {
		greet, err := args.ArgDefault(0, "World")
		if err != nil {
			return err
		}
		fmt.Println(fmt.Sprintf("Hello, %s!", greet))
		return nil
	})

	ad.ProvideCommands(helloCmd)

	return ad
}
