// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/x/cli"
	"github.com/mkungla/happy/x/happyx"
	"github.com/mkungla/happy/x/pkg/vars"
)

func Info(opts ...happy.OptionSetFunc) happy.Command {
	cmd, err := cli.NewCommand(
		"info",
		happyx.ReadOnlyOption("usage.decription", "print application info."),
		happyx.ReadOnlyOption("category", "internal"),
	)
	if err != nil {
		return nil
	}
	cmd.Do(func(session happy.Session, flags happy.Flags, assets happy.FS, status happy.ApplicationStatus) error {

		var env bytes.Buffer
		env.WriteString("\nINFO\n==========================\n")

		env.WriteString("\nADDONS\n==========================\n")
		for _, ainfo := range status.Addons() {
			ainforow := fmt.Sprintf("%s (%s) - %s\n", ainfo.Name, ainfo.Slug, ainfo.Version)
			env.WriteString(ainforow)
		}

		env.WriteString("\nDEPENDENCIES\n==========================\n")
		for _, dep := range status.Dependencies() {
			env.WriteString(dep.Path + "\n" + dep.Version + " - " + dep.Sum + "\n\n")
		}

		env.WriteString("\nDEBUG INFO\n==========================\n")
		var (
			debugInfoKeys       []string
			longestDebugInfoKey int
		)

		debugInfoVars := make(map[string]string)
		status.DebugInfo().Range(func(v happy.Variable) bool {
			kind := vars.ValueKindOf(v.Underlying())
			debugInfoVars[v.Key()] = fmt.Sprintf("%-8s = %s\n", "("+kind.String()+")", v.String())
			debugInfoKeys = append(debugInfoKeys, v.Key())

			if len(v.Key()) > longestDebugInfoKey {
				longestDebugInfoKey = len(v.Key())
			}
			return true
		})
		debugInfoRowTmpl := fmt.Sprintf("%%-%ds%%s", longestDebugInfoKey+1)
		sort.Strings(debugInfoKeys)
		for _, k := range debugInfoKeys {
			env.WriteString(fmt.Sprintf(debugInfoRowTmpl, k, debugInfoVars[k]))
		}

		env.WriteString("\nSERVICES\n==========================\n")
		for _, svc := range status.Services() {
			info := fmt.Sprintf("running(%T), failed(%T), err(%s)", svc.Running, svc.Failed, svc.Err)
			env.WriteString(svc.URL + "\n" + info + "\n")

		}

		env.WriteString("\nSESSION\n==========================\n")
		var (
			sessionKeys       []string
			longestSessionKey int
		)

		sessionVars := make(map[string]string)
		session.Opts().Range(func(v happy.Variable) bool {
			kind := vars.ValueKindOf(v.Underlying())
			sessionVars[v.Key()] = fmt.Sprintf("%-8s = %s\n", "("+kind.String()+")", v.String())
			sessionKeys = append(sessionKeys, v.Key())
			if len(v.Key()) > longestSessionKey {
				longestSessionKey = len(v.Key())
			}
			return true
		})
		sessionRowTmpl := fmt.Sprintf("%%-%ds%%s", longestSessionKey+1)
		sort.Strings(sessionKeys)
		for _, k := range sessionKeys {
			env.WriteString(fmt.Sprintf(sessionRowTmpl, k, sessionVars[k]))
		}

		env.WriteString("\nSETTINGS\n==========================\n")
		var (
			settingKeys        []string
			longestSettingsKey int
		)
		settingsVars := make(map[string]string)
		session.Settings().Range(func(v happy.Variable) bool {
			kind := vars.ValueKindOf(v.Underlying())
			settingsVars[v.Key()] = fmt.Sprintf("%-8s = %s\n", "("+kind.String()+")", v.String())
			settingKeys = append(settingKeys, v.Key())

			if len(v.Key()) > longestSettingsKey {
				longestSettingsKey = len(v.Key())
			}
			return true
		})

		settingsRowTmpl := fmt.Sprintf("%%-%ds%%s", longestSettingsKey+1)
		sort.Strings(settingKeys)
		for _, k := range settingKeys {
			env.WriteString(fmt.Sprintf(settingsRowTmpl, k, settingsVars[k]))
		}

		fmt.Fprintln(os.Stdout, env.String())

		return nil
	})
	return cmd
}
