// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package commands

import (
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/sdk/options"
)

func Info() *happy.Command {
	cmd := happy.NewCommand("info",
		happy.Option("description", "Display information about application"),
	)
	cmd.Do(func(sess *happy.Session, args happy.Args) error {

		settbl := textfmt.Table{
			Title:      "APPLICATION SETTINGS",
			WithHeader: true,
		}
		settbl.AddRow("KEY", "KIND", "IS SET", "PERSISTENT", "MUTABILITY", "VALUE", "DESCRIPTION")
		for _, s := range sess.Settings().All() {
			settbl.AddRow(s.Key(), s.Kind().String(), fmt.Sprint(s.IsSet()), fmt.Sprint(s.Persistent()), fmt.Sprint(s.Mutability()), s.Value().String(), s.Description())

		}
		fmt.Println(settbl.String())

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

		// sess.Log().Debug("CONFIG")
		// sess.Config().Range(func(v vars.Variable) bool {
		// 	sess.Log().Debug(v.Name(), slog.String("value", v.String()))
		// 	return true
		// })
		return nil
	})
	return cmd
}
