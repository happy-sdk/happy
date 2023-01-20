// Copyright 2022 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package commands

import (
	"sort"

	"github.com/mkungla/happy"
)

func Info() *happy.Command {
	cmd := happy.NewCommand(
		"info",
		happy.Option("description", "application information, config and settings"),
		happy.Option("category", "GENERAL"),
		happy.Option("skip.addons", true),
	)

	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		// RUNTIME
		opts := sess.RuntimeOpts()
		if opts == nil {
			sess.Log().Warn("no runtime options")
			return nil
		}

		olist := opts.All()
		sort.Slice(olist, func(i, j int) bool {
			return olist[j].Name() > olist[i].Name()
		})

		sess.Log().Println("------------ RUNTIME OPTIONS --------------")
		for _, opt := range olist {
			sess.Log().Println("", opt)
		}
		// CONFIG
		config := sess.Config()
		if config == nil {
			sess.Log().Warn("no config")
			return nil
		}

		clist := config.All()
		sort.Slice(clist, func(i, j int) bool {
			return clist[j].Name() > clist[i].Name()
		})

		sess.Log().Println("------------ CONFIG --------------")
		for _, conf := range clist {
			sess.Log().Println("", conf)
		}

		// SETTINGS
		settings := sess.Settings()
		if settings == nil {
			sess.Log().Warn("no settings")
			return nil
		}

		slist := settings.All()
		sort.Slice(slist, func(i, j int) bool {
			return slist[j].Name() > slist[i].Name()
		})

		sess.Log().Println("------------ SETTINGS --------------")
		for _, setting := range slist {
			sess.Log().Println("", setting)
		}

		return nil
	})
	return cmd
}
