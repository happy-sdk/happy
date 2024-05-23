// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package commands

import (
	"fmt"
	"slices"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/options"
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

	return cmd
}
