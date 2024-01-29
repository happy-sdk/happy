// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package commands

import "github.com/happy-sdk/happy"

func Config() *happy.Command {
	cmd := happy.NewCommand("config",
		happy.Option("description", "configure Happy-SDK application"),
	)
	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		sess.Log().NotImplemented("cmdconfig.Command.Do")
		return nil
	})
	return cmd
}
