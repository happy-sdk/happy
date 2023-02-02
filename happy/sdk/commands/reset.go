// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package commands

import (
	"fmt"
	"os"

	"github.com/mkungla/happy"
)

func Reset() *happy.Command {
	cmd := happy.NewCommand(
		"reset",
		happy.Option("usage", "reset application information, warning it would remove all settings and user data"),
		happy.Option("category", "GENERAL"),
		happy.Option("allow.on.fresh.install", false),
		happy.Option("skip.addons", true),
	)

	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		if !sess.Get("app.fs.enabled").Bool() {
			return fmt.Errorf("%w: app.fs.enabled is false", happy.ErrApplication)
		}
		// cache
		cache := sess.Get("app.path.cache").String()
		if len(cache) == 0 {
			return fmt.Errorf("%w: app.path.cache is empty", happy.ErrApplication)
		}
		if err := os.RemoveAll(cache); err != nil {
			return err
		}
		// config
		config := sess.Get("app.path.config").String()
		if len(config) == 0 {
			return fmt.Errorf("%w: app.path.config is empty", happy.ErrApplication)
		}
		if err := os.RemoveAll(config); err != nil {
			return err
		}
		sess.Log().Ok("application reset")
		return nil
	})
	return cmd
}
