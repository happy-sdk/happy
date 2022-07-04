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

// Package config enables you to configure happy application instance.
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/mkungla/happy"
	"github.com/mkungla/happy/cli"
	"github.com/mkungla/vars/v5"
	"golang.org/x/exp/slices"
)

func Env() (happy.Command, error) {
	cmd, err := cli.NewCommand("env", 2)
	if err != nil {
		return nil, err
	}

	cmd.SetCategory("SYSTEM")
	cmd.SetShortDesc("app environment helper command")

	cmd.Before(func(ctx happy.Session) error {
		ctx.Log().SetLevel(happy.LevelError)
		return nil
	})
	cmd.Do(func(ctx happy.Session) error {
		if len(ctx.Args()) == 0 {
			printEnv(ctx)
			return nil
		}
		if !slices.Contains(
			[]string{"session", "settings"}, ctx.Args()[0].String(),
		) {
			return fmt.Errorf("missing arg %s [session|settings]", ctx.Args()[0])
		}

		if len(ctx.Args()) < 3 || !slices.Contains(
			[]string{"get", "has"}, ctx.Args()[1].String(),
		) {
			return errors.New("missing arg [get|has] key")
		}

		var err error
		if ctx.Args()[0].String() == "settings" {
			err = envSettingAction(ctx, ctx.Args()[1].String(), ctx.Args()[2].String())
		} else {

		}

		return err
	})

	return cmd, nil
}

func envSettingAction(ctx happy.Session, action, key string) error {
	switch action {
	case "get":
		ctx.Out(ctx.Settings().Get(key).Raw())
	case "has":
		ctx.Out(ctx.Settings().Has(key))
	}
	return nil
}

func printEnv(ctx happy.Session) {
	f, err := ctx.Flags().Get("json")
	if err == nil && f.Present() {
		res := EnvResponse{
			Session:  make(map[string]any),
			Settings: make(map[string]any),
		}
		ctx.Range(func(key string, val vars.Value) bool {
			res.Session[key] = val.Raw()
			return true
		})
		ctx.Settings().Range(func(key string, val vars.Value) bool {
			res.Settings[key] = val.Raw()
			return true
		})

		ctx.Out(res)
	} else {
		var (
			sessionKeys        []string
			settingKeys        []string
			longestSessionKey  int
			longestSettingsKey int
		)
		sessionVars := make(map[string]string)
		settings := make(map[string]string)

		ctx.Range(func(key string, val vars.Value) bool {
			sessionVars[key] = fmt.Sprintf("%-8s = %s\n", "("+val.Type().String()+")", val.String())
			sessionKeys = append(sessionKeys, key)
			if len(key) > longestSessionKey {
				longestSessionKey = len(key)
			}
			return true
		})
		ctx.Settings().Range(func(key string, val vars.Value) bool {
			settings[key] = fmt.Sprintf("%-8s = %s\n", "("+val.Type().String()+")", val.String())
			settingKeys = append(settingKeys, key)

			if len(key) > longestSettingsKey {
				longestSettingsKey = len(key)
			}
			return true
		})
		sessionRowTmpl := fmt.Sprintf("%%-%ds%%s", longestSessionKey+1)
		sort.Strings(sessionKeys)
		settingsRowTmpl := fmt.Sprintf("%%-%ds%%s", longestSettingsKey+1)
		sort.Strings(settingKeys)

		var env bytes.Buffer
		env.WriteString("\nSESSION\n")
		for _, k := range sessionKeys {
			env.WriteString(fmt.Sprintf(sessionRowTmpl, k, sessionVars[k]))
		}
		env.WriteString("\nSETTINGS\n")
		for _, k := range settingKeys {
			env.WriteString(fmt.Sprintf(settingsRowTmpl, k, settings[k]))
		}

		fmt.Fprintln(os.Stdout, env.String())
	}
}

type EnvResponse struct {
	Session  map[string]any `json:"session"`
	Settings map[string]any `json:"settings"`
}
