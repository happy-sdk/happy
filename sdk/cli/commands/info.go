// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package commands

import (
	"fmt"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
)

func Info() *happy.Command {
	cmd := happy.NewCommand("info",
		happy.Option("description", "Display information about application"),
	)

	cmd.Do(func(sess *happy.Session, args happy.Args) error {
		infoTbl := textfmt.Table{
			Title: "APPLICATION INFO",
		}

		infoTbl.AddRow("Name", sess.Get("app.name").String())
		infoTbl.AddRow("Slug", sess.Get("app.slug").String())
		infoTbl.AddRow("Description", sess.Get("app.description").String())
		infoTbl.AddRow("Copyright by", sess.Get("app.copyright_by").String())
		infoTbl.AddRow("Copyright since", sess.Get("app.copyright_since").String())
		infoTbl.AddRow("License", sess.Get("app.license").String())
		infoTbl.AddRow("Identifier", sess.Get("app.identifier").String())
		infoTbl.AddRow("", "")
		infoTbl.AddRow("Version", sess.Get("app.version").String())

		fmt.Println(infoTbl.String())

		return nil
	})
	return cmd
}
