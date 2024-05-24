// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package commands

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
)

func Config() *happy.Command {
	cmd := happy.NewCommand("config",
		happy.Option("description", "Application configuration tools"),
	)
	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		// Application info
		infoiFields := []string{
			"app.name",
			"app.slug",
			"app.identifier",
			"app.description",
			"app.copyright_by",
			"app.copyright_since",
			"app.license",
		}
		infotbl := textfmt.Table{
			Title:      "APPLICATION INFO",
			WithHeader: true,
		}
		infotbl.AddRow("KEY", "KIND", "IS SET", "MUTABILITY", "VALUE")

		for _, k := range infoiFields {
			s := sess.Settings().Get(k)
			infotbl.AddRow(s.Key(), s.Kind().String(), fmt.Sprint(s.IsSet()), fmt.Sprint(s.Mutability()), s.Value().String())
		}
		fmt.Println(infotbl.String())

		stbl := textfmt.Table{
			Title:      "APPLICATION SETTINGS",
			WithHeader: true,
		}
		stbl.AddRow("KEY", "KIND", "IS SET", "MUTABILITY", "VALUE", "DESCRIPTION")
		for _, s := range sess.Settings().All() {
			if s.Persistent() || slices.Contains(infoiFields, s.Key()) {
				continue
			}
			stbl.AddRow(s.Key(), s.Kind().String(), fmt.Sprint(s.IsSet()), fmt.Sprint(s.Mutability()), s.Value().String(), s.Description())
		}
		fmt.Println(stbl.String())

		ptbl := textfmt.Table{
			Title:      fmt.Sprintf("PROFILE SETTINGS FOR %s", sess.Get("app.profile.name").String()),
			WithHeader: true,
		}
		ptbl.AddRow("KEY", "KIND", "IS SET", "MUTABILITY", "VALUE", "DESCRIPTION")
		for _, s := range sess.Settings().All() {
			if !s.Persistent() {
				continue
			}
			ptbl.AddRow(s.Key(), s.Kind().String(), fmt.Sprint(s.IsSet()), fmt.Sprint(s.Mutability()), s.Value().String(), s.Description())
		}
		fmt.Println(ptbl.String())

		opttbl := textfmt.Table{
			Title:      "APPLICATION OPTIONS",
			WithHeader: true,
		}
		opttbl.AddRow("KEY", "KIND", "READONLY", "VALUE")
		sess.Opts().Range(func(opt options.Option) bool {
			opttbl.AddRow(opt.Name(), opt.Kind().String(), fmt.Sprintf("%t", opt.ReadOnly()), opt.Value().String())
			return true
		})
		fmt.Println(opttbl.String())

		return nil
	})

	cmd.AddSubCommand(configSet())
	return cmd
}

func configSet() *happy.Command {
	cmd := happy.NewCommand("set",
		happy.Option("description", "Set application configuration values"),
		happy.Option("argn.min", 2),
		happy.Option("argn.max", 2),
		happy.Option("firstuse.allowed", true),
	)
	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		if args.Argn() != 2 {
			return fmt.Errorf("expecting exactly 2 arguments key value")
		}

		key := args.Arg(0).String()
		val := args.Arg(1).String()

		for _, s := range sess.Settings().All() {
			if !s.Persistent() {
				continue
			}
			if s.Key() == key {
				if err := sess.Settings().Set(key, settings.String(val)); err != nil {
					return err
				}
				sess.Log().Ok("setting updated", slog.String("key", key), slog.String("val", val))
				return nil
			}
		}

		return fmt.Errorf("setting key %q is not persistent setting", key)
	})

	cmd.AfterAlways(func(sess *happy.Session, err error) error {
		fmt.Println("AfterAlways: ", sess.Get("app.stats.enabled"))
		return nil
	})
	return cmd
}
