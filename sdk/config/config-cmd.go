// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package config

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/happy-sdk/happy/pkg/options"
	"github.com/happy-sdk/happy/pkg/settings"
	"github.com/happy-sdk/happy/pkg/strings/textfmt"
	"github.com/happy-sdk/happy/pkg/vars"
	"github.com/happy-sdk/happy/pkg/vars/varflag"
	"github.com/happy-sdk/happy/sdk/action"
	"github.com/happy-sdk/happy/sdk/cli/command"
	"github.com/happy-sdk/happy/sdk/internal"
	"github.com/happy-sdk/happy/sdk/session"
)

func Command() *command.Command {
	cmd := command.New(command.Config{
		Name:             "config",
		Category:         "Configuration",
		Description:      "Manage and configure application settings",
		Immediate:        true,
		SkipSharedBefore: true,
	})

	cmd.AddInfo("This command allows you to manage the application configuration settings and settings profiles.")

	cmd.WithSubCommands(
		configLs(),
		configOpts(),
		configSet(),
		configGet(),
		configReset(),
	)

	return cmd
}

func configLs() *command.Command {
	cmd := command.New(command.Config{
		Name:        "ls",
		Description: "List settings for current profile",
		Usage:       "[-a|--all]",
	})

	cmd.Usage("--profile=<profile-name> [flags]")

	cmd.WithFlags(
		varflag.BoolFunc("all", false, "List all settings, including internal settings", "a"),
		varflag.BoolFunc("describe", false, "Describe all displayed settings", "d"),
	)

	cmd.Do(func(sess *session.Context, args action.Args) error {
		var (
			appSettings     []settings.Setting
			profileSettings []settings.Setting
		)

		for _, s := range sess.Settings().All() {
			if !s.Persistent() && !s.UserDefined() {
				appSettings = append(appSettings, s)
				continue
			}
			profileSettings = append(profileSettings, s)
		}

		// Descriptions
		if args.Flag("describe").Var().Bool() {
			desctable := textfmt.Table{
				Title:      "Settings Descriptions",
				WithHeader: true,
			}
			desctable.AddRow("KEY", "DESCRIPTION")
			for _, s := range profileSettings {
				desctable.AddRow(s.Key(), sess.Describe(s.Key()))
			}
			if args.Flag("all").Var().Bool() {
				for _, s := range appSettings {
					desctable.AddRow(s.Key(), sess.Describe(s.Key()))
				}
			}

			sess.Log().Println(desctable.String())
			return nil
		}

		// Profile settings
		table := textfmt.Table{
			Title:      fmt.Sprintf("Settings for current PROFILE: %s", sess.Settings().Name()),
			WithHeader: true,
		}
		table.AddRow("KEY", "KIND", "IS SET", "MUTABILITY", "VALUE", "DEFAULT")
		for _, s := range profileSettings {
			var defval string
			if s.Default().String() != s.Value().String() {
				defval = s.Default().String()
			}
			table.AddRow(s.Key(), s.Kind().String(), fmt.Sprint(s.IsSet()), fmt.Sprint(s.Mutability()), s.Value().String(), defval)
		}
		sess.Log().Println(table.String())

		// App settings
		if !args.Flag("all").Var().Bool() {
			return nil
		}
		apptable := textfmt.Table{
			Title:      "Application Settings (internal)",
			WithHeader: true,
		}

		apptable.AddRow("KEY", "KIND", "IS SET", "MUTABILITY", "VALUE", "DEFAULT")

		for _, s := range appSettings {
			if s.Persistent() || s.UserDefined() {
				appSettings = append(appSettings, s)
				continue
			}
			var defval string
			if s.Mutability() != settings.SettingImmutable && s.Default().String() != s.Value().String() {
				defval = s.Default().String()
			}
			apptable.AddRow(s.Key(), s.Kind().String(), fmt.Sprint(s.IsSet()), fmt.Sprint(s.Mutability()), s.Value().String(), defval)
		}
		sess.Log().Println(apptable.String())

		return nil
	})

	return cmd
}

func configOpts() *command.Command {
	cmd := command.New(command.Config{
		Name:        "opts",
		Description: "List application session options for current profile",
	})

	cmd.Usage("--profile=<profile-name>")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		optstbl := textfmt.Table{}
		sess.Opts().Range(func(opt options.Option) bool {
			optstbl.AddRow(opt.Name(), sess.Describe(opt.Name()), opt.Value().String())
			return true
		})
		sess.Log().Println(optstbl.String())
		return nil
	})

	return cmd
}

func configSet() *command.Command {
	cmd := command.New(command.Config{
		Name:        "set",
		Description: "Set a setting value",
		MinArgs:     2,
	})

	cmd.Usage("--profile=<profile-name>")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := args.Arg(0).String()
		if !sess.Settings().Has(key) {
			return fmt.Errorf("setting %q does not exist", key)
		}
		value := args.Arg(1).String()

		if err := sess.Settings().Validate(key, value); err != nil {
			return err
		}

		profileFilePath := filepath.Join(sess.Get("app.fs.path.profile").String(), "profile.preferences")

		profile := sess.Settings().All()
		pd := vars.Map{}
		for _, setting := range profile {
			if setting.Persistent() || setting.UserDefined() {
				if setting.Key() == key {
					if err := pd.Store(setting.Key(), value); err != nil {
						return err
					}
				} else if setting.IsSet() {
					if err := pd.Store(setting.Key(), setting.Value().String()); err != nil {
						return err
					}
				}
			}
		}
		pddata := pd.ToKeyValSlice()
		var dest bytes.Buffer
		enc := gob.NewEncoder(&dest)
		if err := enc.Encode(pddata); err != nil {
			return err
		}

		internal.Log(sess.Log(), "profile.save",
			slog.String("profile", sess.Get("app.profile.name").String()),
			slog.String("file", profileFilePath),
		)
		if err := os.WriteFile(profileFilePath, dest.Bytes(), 0600); err != nil {
			return err
		}

		internal.Log(
			sess.Log(),
			"saved profile",
			slog.String("profile", sess.Get("app.profile.name").String()),
			slog.String("file", profileFilePath),
		)
		return nil
	})

	return cmd
}

func configGet() *command.Command {
	cmd := command.New(command.Config{
		Name:        "get",
		Description: "Get a setting or option value",
		MinArgs:     1,
	})

	cmd.Usage("--profile=<profile-name>")

	cmd.Do(func(sess *session.Context, args action.Args) error {
		key := sess.Get(args.Arg(0).String())
		if key != vars.EmptyVariable {
			fmt.Println(key.String())
		}
		return nil
	})

	return cmd
}

func configReset() *command.Command {
	cmd := command.New(command.Config{
		Name:        "reset",
		Description: "Reset a setting to its default value",
		MinArgs:     1,
	})

	cmd.Usage("--profile=<profile-name>")

	cmd.WithFlags(varflag.BoolFunc("all", false, "reset all settings", "a"))

	cmd.Do(func(sess *session.Context, args action.Args) error {
		if args.Flag("all").Present() {
			profileFilePath := filepath.Join(sess.Get("app.fs.path.profile").String(), "profile.preferences")
			internal.Log(sess.Log(), "profile.save",
				slog.String("profile", sess.Get("app.profile.name").String()),
				slog.String("file", profileFilePath),
			)

			if err := os.WriteFile(profileFilePath, []byte{}, 0600); err != nil {
				return err
			}

			internal.Log(
				sess.Log(),
				"saved profile",
				slog.String("profile", sess.Get("app.profile.name").String()),
				slog.String("file", profileFilePath),
			)
			return nil
		}

		key := args.Arg(0).String()
		if !sess.Settings().Has(key) {
			return fmt.Errorf("setting %q does not exist", key)
		}

		profileFilePath := filepath.Join(sess.Get("app.fs.path.profile").String(), "profile.preferences")
		internal.Log(sess.Log(), "profile.save",
			slog.String("profile", sess.Get("app.profile.name").String()),
			slog.String("file", profileFilePath),
		)

		profile := sess.Settings().All()
		pd := vars.Map{}
		for _, setting := range profile {
			if setting.Persistent() || setting.UserDefined() {
				if setting.Key() == key {
					continue
				} else if setting.IsSet() {
					if err := pd.Store(setting.Key(), setting.Value().String()); err != nil {
						return err
					}
				}
			}
		}
		pddata := pd.ToKeyValSlice()
		var dest bytes.Buffer
		enc := gob.NewEncoder(&dest)
		if err := enc.Encode(pddata); err != nil {
			return err
		}

		if err := os.WriteFile(profileFilePath, dest.Bytes(), 0600); err != nil {
			return err
		}

		internal.Log(
			sess.Log(),
			"saved profile",
			slog.String("profile", sess.Get("app.profile.name").String()),
			slog.String("file", profileFilePath),
		)
		return nil
	})

	return cmd
}
