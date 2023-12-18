// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package commands

import (
	"fmt"
	"os"

	"github.com/happy-sdk/happy"
)

func Reset() *happy.Command {
	cmd := happy.NewCommand(
		"reset",
		happy.Option("usage", "reset application information, warning it would remove all settings and user data"),
		happy.Option("category", "GENERAL"),
		happy.Option("init.allowed", false),
		happy.Option("addons.disabled", true),
	)

	cmd.Do(func(sess *happy.Session, args happy.Args) error {

		// cache
		cache := sess.Get("app.fs.path.cache").String()
		if len(cache) == 0 {
			return fmt.Errorf("%w: app.fs.path.cache is empty", happy.ErrApplication)
		}
		if err := os.RemoveAll(cache); err != nil {
			return err
		}
		// config
		config := sess.Get("app.fs.path.config").String()
		if len(config) == 0 {
			return fmt.Errorf("%w: app.fs.path.config is empty", happy.ErrApplication)
		}
		if err := os.RemoveAll(config); err != nil {
			return err
		}
		sess.Log().Ok("application reset")
		return nil
	})
	return cmd
}
