// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

package main

import (
	"fmt"

	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/version"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/session"
)

func (s *state) cmdProjectInfo() *command.Command {
	return command.New("info",
		command.Config{
			Description: "Print info about current project",
			Category:    "project",
		}).
		Disable(func(sess *session.Context) error {
			if !s.Found() {
				return fmt.Errorf("project not found")
			}
			return nil
		}).
		Do(func(sess *session.Context, args action.Args) error {
			prj, err := s.Project()
			if err != nil {
				return err
			}
			infotbl := textfmt.NewTable(textfmt.TableWithHeader())

			dir := prj.Dir()

			infotbl.AddRow("dir.path", dir.Path)
			infotbl.AddRow("dir.has_config_file", fmt.Sprint(dir.HasConfigFile))
			infotbl.AddRow("dir.config_file", dir.ConfigFile)
			infotbl.AddRow("dir.happy_version", dir.HappyVersion.String())
			infotbl.AddRow("dir.version", dir.Version.String())
			infotbl.AddRow("dir.depends_on_happy", fmt.Sprint(dir.DependsOnHappy))
			infotbl.AddRow("dir.has_git", fmt.Sprint(dir.HasGit))

			cnf := prj.Config()

			infotbl.AddDivider()
			verstr := cnf.Version().String()
			if version.Compare(version.Version(verstr), version.Version(cnf.Get("version").String())) != 0 {
				verstr = fmt.Sprintf("file schema %s, parsed schema %s", verstr, cnf.Get("version").String())
			}
			infotbl.AddRow("CONFIG", fmt.Sprintf("Schema version: %s", verstr))
			infotbl.AddDivider()
			infotbl.AddRow("KEY", "VALUE", "KIND", "IS SET", "DEFAULT")
			infotbl.AddDivider()
			for c := range cnf.All() {
				infotbl.AddRow(fmt.Sprintf("config.%s", c.Key()), c.Value().String(), c.Kind().String(), fmt.Sprint(c.IsSet()), truncate(c.Default().String(), 12))
			}

			fmt.Println(infotbl.String())

			return nil
		})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
