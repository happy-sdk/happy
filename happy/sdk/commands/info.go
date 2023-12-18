// Copyright 2022 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

// Package commands provides commonly used commands which you can simply plug into your application.
package commands

import (
	"fmt"
	"sort"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/settings"
)

func Info() *happy.Command {
	cmd := happy.NewCommand(
		"info",
		happy.Option("description", "Info displays information about the application and current user environment"),
		happy.Option("usage", "application information, config and settings"),
		happy.Option("category", "GENERAL"),
		happy.Option("addons.disabled", true),
	)

	cmd.Before(func(sess *happy.Session, args happy.Args) error {
		sess.Set("application.info", "collected")
		return nil
	})
	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		// RUNTIME
		opts := sess.Opts()
		if opts == nil {
			fmt.Println("\n----------------- RUNTIME OPTIONS -----------------")
			sess.Log().Notice("no runtime options")
		} else {
			olist := opts.All()
			olistKeyLen := 0
			sort.Slice(olist, func(i, j int) bool {
				if l := len(olist[j].Name()); l > olistKeyLen {
					olistKeyLen = l
				}
				if l := len(olist[i].Name()); l > olistKeyLen {
					olistKeyLen = l
				}
				return olist[j].Name() > olist[i].Name()
			})

			fmt.Println("\n----------------- RUNTIME OPTIONS -----------------")
			optsfmt := fmt.Sprintf("%%-%ds %%-10s %%s\n", olistKeyLen+1)
			for _, opt := range olist {
				fmt.Printf(optsfmt, opt.Name(), opt.Kind(), opt.String())
			}

		}

		// CONFIG
		config := sess.Config()
		if config == nil {
			fmt.Println("\n----------------- RUNTIME CONFIG -----------------")
			sess.Log().Notice("no config")
		} else {
			clist := config.All()
			clistKeyLen := 0
			sort.Slice(clist, func(i, j int) bool {
				if l := len(clist[j].Name()); l > clistKeyLen {
					clistKeyLen = l
				}
				if l := len(clist[i].Name()); l > clistKeyLen {
					clistKeyLen = l
				}
				return clist[j].Name() > clist[i].Name()
			})

			fmt.Println("\n----------------- RUNTIME CONFIG -----------------")
			conffmt := fmt.Sprintf("%%-%ds %%-10s %%s\n", clistKeyLen+1)
			for _, conf := range clist {
				fmt.Printf(conffmt, conf.Name(), conf.Kind(), conf.String())
			}
		}

		// SETTINGS
		profile := sess.Profile()
		var (
			slist       []settings.Setting
			slistKeyLen int
		)
		if profile == nil {
			fmt.Println("\n----------------- USER SETTINGS -----------------")
			sess.Log().Notice("no settings profile")
		} else {
			slist = profile.All()
			slistKeyLen = 0
			sort.Slice(slist, func(i, j int) bool {
				if l := len(slist[j].Key()); l > slistKeyLen {
					slistKeyLen = l
				}
				if l := len(slist[i].Key()); l > slistKeyLen {
					slistKeyLen = l
				}
				return slist[j].Key() > slist[i].Key()
			})
			fmt.Println("\n----------------- USER SETTINGS -----------------")
			fmt.Println("PROFILE:", profile.Name())
			fmt.Println("VERSION:", profile.Version())
			fmt.Println("PKG:", profile.Pkg())
			fmt.Println("MODULE:", profile.Module())
			fmt.Println("-------------------------------------------------")
			settingfmt := fmt.Sprintf("%%-%ds %%-10s %%s\n", slistKeyLen+1)
			for _, setting := range slist {
				fmt.Printf(settingfmt, setting.Key(), setting.Kind(), setting.String())
			}
		}

		return nil
	})
	return cmd
}
