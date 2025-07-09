// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2025 The Happy Authors

package projects

import (
	"fmt"
	"path"

	"github.com/happy-sdk/happy"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func cmdProjectInfo() *command.Command {
	return command.New("info",
		command.Config{
			Description: "Print info about current project",
		}).
		Do(func(sess *session.Context, args action.Args) error {
			api, err := happy.API[*API](sess)
			if err != nil {
				return err
			}
			prj, err := api.Project(sess, true)
			if err != nil {
				return err
			}

			info := textfmt.Table{
				Title:      "Project Information",
				WithHeader: true,
			}
			info.AddRow("key", "value")
			for opt := range prj.Config().All() {
				info.AddRow(opt.Key(), opt.String())
			}
			fmt.Println(info.String())

			gomodules, err := prj.GoModules(sess, true, false)
			if err != nil {
				return err
			}
			modulelist := textfmt.Table{
				Title: "Packages",
			}
			modulelist.AddRow(
				"Package",
				"Action",
				"Current",
				"Next",
				"Update deps",
			)
			for _, pkg := range gomodules {
				action := "skip"
				if pkg.NeedsRelease {
					action = "release"
				}
				if pkg.FirstRelease {
					action = "initial"
				}
				modulelist.AddRow(
					pkg.Import,
					action,
					path.Base(pkg.LastReleaseTag),
					path.Base(pkg.NextReleaseTag),
					fmt.Sprint(pkg.UpdateDeps),
				)
			}
			fmt.Println(modulelist.String())
			return nil
		})
}
